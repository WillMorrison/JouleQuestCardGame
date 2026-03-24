import numpy as np

from gymnasium import spaces
from gymnasium.utils import seeding

from pettingzoo import AECEnv
from pettingzoo.utils import AgentSelector

from apiclient import joule_quest_api_client
from apiclient.joule_quest_api_client.models import PlayerAction, PlayerActionAssetType, PlayerActionType, GameStatus, GameReason, PlayerStatus
from game_client import GameClient

# The observation space for a single player
OBSERVATION_SPACE = spaces.Dict({
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
           "action_mask": spaces.Discrete(15, dtype=np.int8)
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

class CustomEnvironment(AECEnv):
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

        self.possible_agents = list(range(num_players))
        self.observation_spaces = {agent: OBSERVATION_SPACE for agent in self.possible_agents}
        self.action_spaces = {agent: ACTION_SPACE for agent in self.possible_agents}

    def reset(self, seed=None, options=None):
        if self._game_client is not None:
            self._game_client.close()
        self._game_client = GameClient(self._api_client, self.max_num_agents)

        # Unlike gymnasium's Env, the environment is responsible for setting the random seed explicitly.
        if seed is not None:
            self.np_random, self.np_random_seed = seeding.np_random(seed)
        self.agents: list[int] = self.possible_agents[:]
        self.rewards = {agent: 0 for agent in self.agents}
        self.terminations = {agent: False for agent in self.agents}
        self.truncations = {agent: False for agent in self.agents}
        self.infos = {agent: {} for agent in self.agents}
        self.observations = {agent: {} for agent in self.agents}
        self._agent_selector = AgentSelector(self.agents)
        self.agent_selection = self._agent_selector.next()

    def close(self):
        if self._game_client is not None:
            self._game_client.close()
        self._game_client = None

    def step(self, action:int):
        if self._game_client is None:
            raise TypeError("Game client should be initialized")

        possible_actions = [a for a in self._game_client.possible_actions if a.player_index==self.agent_selection]
        if not possible_actions:
            # Chosen agent can't do anything, move along
            self.agent_selection = self._agent_selector.next()
            return
        chosen_action = IntToPlayerAction(action, possible_actions)

        self._game_client.send_action(chosen_action)

        # Handle global game end conditions
        if self._game_client.game.status == GameStatus.LOSS:
            for a in self.agents:
                self.rewards[a] -= 1000  # Large collective penalty
                self.terminations[a] = True
        elif self._game_client.game.status == GameStatus.WIN:
            for a in self.agents:
                self.rewards[a] += 100 # Reward for Winning!
                self.rewards[a] += self._game_client.game.players[a].money # Reward for successful capitalism
                self.terminations[a] = True

        player = self._game_client.game.players[self.agent_selection]
        if player.status == PlayerStatus.LOST:
            self.rewards[self.agent_selection] -= 1000 # Large penalty for losing
            self.terminations[self.agent_selection] = True
        else:
            self.rewards[self.agent_selection] += 0.01*player.money # Hint that more money is good
            self.rewards[self.agent_selection] -= 0.01*self._game_client.game.emissions_counter # Hint that emissions counter going up is bad

        self.agent_selection = self._agent_selector.next()


    def observe(self, agent:int):
        if self._game_client is None:
            raise TypeError("Game client should be initialized")
        
        # Get valid action ints for agent (e.g., [0, 4, 7])
        valid_actions = [PlayerActionToInt(a) for a in self._game_client.possible_actions if a.player_index==agent]
        
        # Create a binary mask of 0s (forbidden) and 1s (allowed)
        mask = [1 if i in valid_actions else 0 for i in range(15)]
        
        return {
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
                "Status": PlayerStatusToInt(self._game_client.game.players[agent].status),
                "Money": self._game_client.game.players[agent].money,
                "AssetMix": self._game_client.game.players[agent].assets.to_dict(),
            },
            "action_mask": mask,
        }

    def render(self):
        pass