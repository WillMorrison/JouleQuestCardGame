import numpy as np

from gymnasium import spaces
from gymnasium.utils import seeding

from pettingzoo import AECEnv
from pettingzoo.utils import wrappers

from apiclient import joule_quest_api_client
from apiclient.joule_quest_api_client.models import PlayerAction, PlayerActionAssetType, PlayerActionType, Game, GameStatus, GameReason, PlayerStatus
from game_client import GameClient

# The observation space for a single player's view of the game
# Flattened to a vector of scalars to avoid large one-hot encodings
OBSERVATION_LOW = np.array([
    0,  # Game_Status
    0,  # Game_Reason
    0,  # Game_Round
    0,  # Game_EmissionsCounter
    0,  # Game_LastAssetMix_Renewables
    0,  # Game_LastAssetMix_BatteriesArbitrage
    0,  # Game_LastAssetMix_BatteriesCapacity
    0,  # Game_LastAssetMix_FossilsWholesale
    0,  # Game_LastAssetMix_FossilsCapacity
    0,  # Game_LastGridStability
    0,  # Game_LastPriceVolatility
    0,  # Player_Status
    -1024,  # Player_Money
    0,  # Player_AssetMix_Renewables
    0,  # Player_AssetMix_BatteriesArbitrage
    0,  # Player_AssetMix_BatteriesCapacity
    0,  # Player_AssetMix_FossilsWholesale
    0,  # Player_AssetMix_FossilsCapacity
], dtype=np.int32)

OBSERVATION_SIZE = np.array([
    2,  # Game_Status
    7,  # Game_Reason
    1023,  # Game_Round
    1023,  # Game_EmissionsCounter
    1023,  # Game_LastAssetMix_Renewables
    1023,  # Game_LastAssetMix_BatteriesArbitrage
    1023,  # Game_LastAssetMix_BatteriesCapacity
    1023,  # Game_LastAssetMix_FossilsWholesale
    1023,  # Game_LastAssetMix_FossilsCapacity
    3,  # Game_LastGridStability
    3,  # Game_LastPriceVolatility
    1,  # Player_Status
    65535, # Player_Money (approx)
    1023,  # Player_AssetMix_Renewables
    1023,  # Player_AssetMix_BatteriesArbitrage
    1023,  # Player_AssetMix_BatteriesCapacity
    1023,  # Player_AssetMix_FossilsWholesale
    1023,  # Player_AssetMix_FossilsCapacity
], dtype=np.int32)

OBSERVATION_SPACE = spaces.Dict({
    "observation": spaces.MultiDiscrete(start=OBSERVATION_LOW, nvec=OBSERVATION_SIZE, dtype=np.int32),
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
            # Chosen agent can't do anything, move along. They probably finished building already.
            self.agent_selection = self._select_active_agent()
            return
        try:
            chosen_action = IntToPlayerAction(action, possible_actions)
        except KeyError:
            # Illegal action chosen, penalize and move along
            self.rewards[self.agent_selection] -= 10
            self.agent_selection = self._select_active_agent()
            self._accumulate_rewards()
            return

        if chosen_action.type_ in (PlayerActionType.SCRAPASSET, PlayerActionType.TAKEOVERSCRAPASSET) and chosen_action.asset_type == PlayerActionAssetType.FOSSIL:
            self.rewards[self.agent_selection] += 1 # Small reward for scrapping fossil assets

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
                self.rewards[a] += 1000 # Reward for Winning!
                self.rewards[a] += self._game_client.game.players[a_i].money # Reward for successful capitalism
                self.terminations[a] = True

        # Handle individual player loss and reward. This is done for all players every round, so as not to
        # unfairly reward players more just because they happen to have been chosen to play more steps.
        for a_i, a in enumerate(self.agents):
            if self.terminations[a]:
                continue
            player = self._game_client.game.players[a_i]

            if player.status == PlayerStatus.LOST:
                self.rewards[a] -= 1000  # Large penalty for losing
                self.terminations[a] = True
            elif player.status == PlayerStatus.ACTIVE:
                # Survival reward to encourage not losing
                self.rewards[a] += 0.1
                # Small reward for accumulating money (encourages capitalism)
                # self.rewards[a] += player.money * 0.01
                # Small cost for holding fossil assets (encourages transition)
                self.rewards[a] -= player.assets.fossils_capacity * 0.01
                self.rewards[a] -= player.assets.fossils_wholesale * 0.01

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
        
        game = self._game_client.game
        player = game.players[agent_index]
        last_snapshot = game.last_round_snapshot
        
        observation = np.array([
            GameStatusToInt(game.status),
            GameReasontoInt(game.reason),
            game.round_,
            game.emissions_counter,
            last_snapshot.asset_mix.renewables,
            last_snapshot.asset_mix.batteries_arbitrage,
            last_snapshot.asset_mix.batteries_capacity,
            last_snapshot.asset_mix.fossils_wholesale,
            last_snapshot.asset_mix.fossils_capacity,
            last_snapshot.grid_stability,
            last_snapshot.price_volatility,
            PlayerStatusToInt(player.status),
            player.money,
            player.assets.renewables,
            player.assets.batteries_arbitrage,
            player.assets.batteries_capacity,
            player.assets.fossils_wholesale,
            player.assets.fossils_capacity,
        ], dtype=np.int32)
        
        return {
            "observation": observation,
            "action_mask": np.array(mask, dtype=np.int8),
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