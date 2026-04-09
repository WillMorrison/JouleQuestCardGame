import numpy as np

from gymnasium import spaces
from gymnasium.utils import seeding

from pettingzoo import AECEnv
from pettingzoo.utils import wrappers

from apiclient import joule_quest_api_client
from apiclient.joule_quest_api_client.models import PlayerAction, PlayerActionAssetType, PlayerActionType, Game, GameStatus, GameReason, PlayerStatus
from game_client import GameClient

# The observation space for a single player's view of the game
OBSERVATION_SPACE_INNER = spaces.Dict({
            "Game": spaces.Dict({
                "Status": spaces.Discrete(3, dtype=np.int8), # Corresponds to engine.GameStatus
                "Reason": spaces.Discrete(8, dtype=np.int8), # Corresponds to engine.LossCondition
                "Round": spaces.Discrete(1024),
                "EmissionsCounter": spaces.Discrete(1024),
                "LastAssetMix": spaces.Dict({
                    "Renewables": spaces.Discrete(1024),
                    "BatteriesArbitrage": spaces.Discrete(1024),
                    "BatteriesCapacity": spaces.Discrete(1024),
                    "FossilsWholesale": spaces.Discrete(1024),
                    "FossilsCapacity": spaces.Discrete(1024),
                }),
                "LastGridStability": spaces.Discrete(4, dtype=np.int8), # Corresponds to core.GridStability
                "LastPriceVolatility": spaces.Discrete(4, dtype=np.int8), # Corresponds to core.PriceVolatility
            }),
            "Player": spaces.Dict({
                "Status": spaces.Discrete(2, dtype=np.int8), # Corresponds to engine.PlayerStatus
                "Money": spaces.Discrete(65535, start=-1024),
                "AssetMix": spaces.Dict({
                    "Renewables": spaces.Discrete(1024),
                    "BatteriesArbitrage": spaces.Discrete(1024),
                    "BatteriesCapacity": spaces.Discrete(1024),
                    "FossilsWholesale": spaces.Discrete(1024),
                    "FossilsCapacity": spaces.Discrete(1024),
                }),
            }),
        })

# The observation space for a single player, including both game state and available actions.
# Game state has been flattened so that it's possible to use for a neural network.
OBSERVATION_SPACE = spaces.Dict({
            "observation": spaces.flatten_space(OBSERVATION_SPACE_INNER),
            "action_mask": spaces.MultiBinary(15)
        })

# (build, scrap, takeover, takeover+scrap)x(renewable, battery, fossil) + (pledge)x(battery, fossil) + (finished)
ACTION_SPACE = spaces.Discrete(15)

def PlayerActionToInt(player_action: PlayerAction) -> int:
    if player_action.type_ == PlayerActionType.BUILDASSET and player_action.asset_type == PlayerActionAssetType.RENEWABLE:
        return 0
    elif player_action.type_ == PlayerActionType.BUILDASSET and player_action.asset_type == PlayerActionAssetType.BATTERY:
        return 1
    elif player_action.type_ == PlayerActionType.BUILDASSET and player_action.asset_type == PlayerActionAssetType.FOSSIL:
        return 2
    elif player_action.type_ == PlayerActionType.SCRAPASSET and player_action.asset_type == PlayerActionAssetType.RENEWABLE:
        return 3
    elif player_action.type_ == PlayerActionType.SCRAPASSET and player_action.asset_type == PlayerActionAssetType.BATTERY:
        return 4
    elif player_action.type_ == PlayerActionType.SCRAPASSET and player_action.asset_type == PlayerActionAssetType.FOSSIL:
        return 5
    elif player_action.type_ == PlayerActionType.TAKEOVERASSET and player_action.asset_type == PlayerActionAssetType.RENEWABLE:
        return 6
    elif player_action.type_ == PlayerActionType.TAKEOVERASSET and player_action.asset_type == PlayerActionAssetType.BATTERY:
        return 7
    elif player_action.type_ == PlayerActionType.TAKEOVERASSET and player_action.asset_type == PlayerActionAssetType.FOSSIL:
        return 8
    elif player_action.type_ == PlayerActionType.TAKEOVERSCRAPASSET and player_action.asset_type == PlayerActionAssetType.RENEWABLE:
        return 9
    elif player_action.type_ == PlayerActionType.TAKEOVERSCRAPASSET and player_action.asset_type == PlayerActionAssetType.BATTERY:
        return 10
    elif player_action.type_ == PlayerActionType.TAKEOVERSCRAPASSET and player_action.asset_type == PlayerActionAssetType.FOSSIL:
        return 11
    elif player_action.type_ == PlayerActionType.PLEDGECAPACITY and player_action.asset_type == PlayerActionAssetType.BATTERY:
        return 12
    elif player_action.type_ == PlayerActionType.PLEDGECAPACITY and player_action.asset_type == PlayerActionAssetType.FOSSIL:
        return 13
    elif player_action.type_ == PlayerActionType.FINISHED:
        return 14
    raise ValueError(f"Invalid Action {player_action}")


def IntToPlayerAction(action: int, possible_actions:list[PlayerAction]) -> PlayerAction:
    for pa in possible_actions:
        if PlayerActionToInt(pa)==action:
            return pa
    raise KeyError(f"action {action} is not possible")

def PlayerStatusToInt(status: PlayerStatus) -> np.int8:
    if status == PlayerStatus.ACTIVE:
        return np.int8(0)
    elif status == PlayerStatus.LOST:
        return np.int8(1)
    raise ValueError(f"Invalid Player Status {status}")

def GameStatusToInt(status: GameStatus) -> np.int8:
    if status == GameStatus.ONGOING:
        return np.int8(0)
    elif status == GameStatus.LOSS:
        return np.int8(1)
    elif status == GameStatus.WIN:
        return np.int8(2)
    raise ValueError(f"Invalid Game Status {status}")

def GameReasontoInt(reason: GameReason) -> np.int8:
    if reason == GameReason.NONE:
        return np.int8(0)
    elif reason == GameReason.CARBONEMISSIONSEXCEEDED:
        return np.int8(1)
    elif reason == GameReason.INSUFFICIENTGENERATION:
        return np.int8(2)
    elif reason == GameReason.GRIDUNSTABLE:
        return np.int8(3)
    elif reason == GameReason.UNOWNEDTAKEOVERASSETS:
        return np.int8(4)
    elif reason == GameReason.NOACTIVEPLAYERS:
        return np.int8(5)
    raise ValueError(f"Invalid Game Loss Reason {reason}")

def _agent_name(i: int)-> str:
    return f"player_{i}"

def env(num_players: int, client: joule_quest_api_client.Client):
    env = JoulequestEnv(num_players, client)
    env = wrappers.AssertOutOfBoundsWrapper(env)
    env = wrappers.OrderEnforcingWrapper(env)
    return env

agentType = str

class JoulequestEnv(AECEnv):
    metadata = {
        "name": "joulequest_environment_v0",
    }

    def __init__(self, num_players: int, client: joule_quest_api_client.Client):
        """Creates a new environment connected to the REST API via a unix socket.

        Args:
            num_players: The number of players to simulate
            client: A client connected to the API.
        """
        self._api_client = client
        self._game_client : GameClient|None = None

        self.possible_agents: list[agentType] = [_agent_name(i) for i in range(num_players)]
        self._agent_index: dict[agentType, int] = {_agent_name(i): i for i in range(num_players)}
        self.observation_spaces = {agent: OBSERVATION_SPACE for agent in self.possible_agents}
        self.action_spaces = {agent: ACTION_SPACE for agent in self.possible_agents}

    @property
    def _active_agents(self) -> list[agentType]:
        if self._game_client is None:
            return []
        return list(set(_agent_name(a.player_index) for a in self._game_client.possible_actions))
    
    def _select_active_agent(self) -> agentType:
        return self.np_random.choice(self._active_agents)

    def reset(self, seed=None, options=None):
        if self._game_client is not None:
            self._game_client.close()
        self._game_client = GameClient(self._api_client, self.max_num_agents)

        # Unlike gymnasium's Env, the environment is responsible for setting the random seed explicitly.
        self.np_random, self.np_random_seed = seeding.np_random(seed)
        self.agents: list[agentType] = self.possible_agents[:]
        self.rewards: dict[agentType, float] = {agent: 0 for agent in self.agents}
        self._cumulative_rewards: dict[agentType, float] = {agent: 0 for agent in self.agents}
        self.terminations: dict[agentType, bool] = {agent: False for agent in self.agents}
        self.truncations: dict[agentType, bool] = {agent: False for agent in self.agents}
        self.infos = {agent: {} for agent in self.agents}
        self.observations = {agent: {} for agent in self.agents}
        self.agent_selection: agentType = self._select_active_agent()

    def step(self, action:int|None):
        if self._game_client is None:
            raise TypeError("Game client should be initialized")
        if (self.terminations[self.agent_selection] or self.truncations[self.agent_selection]):
            self._was_dead_step(action)
            return
        if action is None:
            self.agent_selection = self._select_active_agent()
            return
        
        # cumulative rewards from previous iterations should be cleared
        # seems weird, but makes the api_test pass
        self._cumulative_rewards[self.agent_selection] = 0

        possible_actions = [a for a in self._game_client.possible_actions if a.player_index==self._agent_index[self.agent_selection]]
        if not possible_actions:
            # Chosen agent can't do anything, move along
            self.agent_selection = self._select_active_agent()
            return
        chosen_action = IntToPlayerAction(action, possible_actions)

        self._game_client.send_action(chosen_action)

        # Handle global game end conditions
        is_over = False
        if self._game_client.game.status == GameStatus.LOSS:
            is_over = True
            for a in self.agents:
                self.rewards[a] -= 1000  # Large collective penalty
                self.terminations[a] = True
        elif self._game_client.game.status == GameStatus.WIN:
            is_over = True
            for a_i, a in enumerate(self.agents):
                self.rewards[a] += 100 # Reward for Winning!
                self.rewards[a] += self._game_client.game.players[a_i].money # Reward for successful capitalism
                self.terminations[a] = True

        # Handle player loss for active agents
        for a_i, a in enumerate(self.agents):
            if self._game_client.game.players[a_i].status == PlayerStatus.LOST:
                self.rewards[a] -= 1000  # Large penalty for losing
                self.terminations[a] = True

        # Incremental rewards if the active agent is still in the game
        player = self._game_client.game.players[self._agent_index[self.agent_selection]]
        if player.status != PlayerStatus.LOST:
            self.rewards[self.agent_selection] += 0.001*player.money # Hint that more money is good
            self.rewards[self.agent_selection] -= 0.001*self._game_client.game.emissions_counter # Hint that emissions counter going up is bad

        if not is_over:
            self.agent_selection = self._select_active_agent()

        self._accumulate_rewards()

    def _action_mask(self, agent_index:int, possible_actions: list[PlayerAction]) -> list[int]:
        # Get valid action ints for agent (e.g., [0, 4, 7])
        valid_actions = [PlayerActionToInt(a) for a in possible_actions if a.player_index==agent_index]
        
        # Create a binary mask of 0s (forbidden) and 1s (allowed)
        mask = [1 if i in valid_actions else 0 for i in range(15)]

        return mask


    def observe(self, agent: agentType):
        if self._game_client is None:
            raise TypeError("Game client should be initialized")
        agent_index = self._agent_index[agent]
        
        mask = self._action_mask(agent_index, self._game_client.possible_actions)
        
        return {
            "observation": spaces.flatten(OBSERVATION_SPACE_INNER, {
                "Game": {
                    "Status": GameStatusToInt(self._game_client.game.status),
                    "Reason": GameReasontoInt(self._game_client.game.reason),
                    "Round": self._game_client.game.round_,
                    "EmissionsCounter": self._game_client.game.emissions_counter,
                    "LastAssetMix": self._game_client.game.last_round_snapshot.asset_mix.to_dict(),
                    "LastGridStability": self._game_client.game.last_round_snapshot.grid_stability,
                    "LastPriceVolatility": self._game_client.game.last_round_snapshot.price_volatility,
                },
                "Player": {
                    "Status": PlayerStatusToInt(self._game_client.game.players[agent_index].status),
                    "Money": self._game_client.game.players[agent_index].money,
                    "AssetMix": self._game_client.game.players[agent_index].assets.to_dict(),
                },
            }),
            "action_mask":np.array(mask, dtype=np.int8),
        }

    def render(self):
        pass

    def close(self):
        super().close()
        if self._game_client is not None:
            self._game_client.close()

    def action_space(self, agent: agentType) -> spaces.Space:
        return ACTION_SPACE
    
    def observation_space(self, agent: agentType) -> spaces.Space:
        return OBSERVATION_SPACE
    
    @property
    def game(self) -> Game:
        if self._game_client is None:
            raise TypeError("Game client should be initialized")
        return self._game_client.game
    
    def get_log(self) -> str:
        if self._game_client is None:
            raise TypeError("Game client should be initialized")
        return self._game_client.get_log()