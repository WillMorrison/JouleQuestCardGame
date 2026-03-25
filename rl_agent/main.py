import argparse
import collections
from concurrent import futures
import contextlib
import itertools
import random
from typing import Final

import numpy as np

from game_client import ServerClient, GameClient
from apiclient import joule_quest_api_client as client
from apiclient.joule_quest_api_client.models import PlayerAction, Game, GameReason, GameStatus, PlayerActionType, PlayerActionAssetType

from custom_environment.env import joulequest_env


UDS_PATH: Final[str] = "/tmp/joulequest_api.sock"


def _is_stupid(action: PlayerAction)->bool:
    if action.type_ is PlayerActionType.BUILDASSET and action.asset_type is PlayerActionAssetType.FOSSIL:
        return True
    if (action.type_ is PlayerActionType.SCRAPASSET or action.type_ is PlayerActionType.TAKEOVERSCRAPASSET) and (action.asset_type is PlayerActionAssetType.BATTERY or action.asset_type is PlayerActionAssetType.RENEWABLE):
        return True

    return False

def filter_stupid_actions(actions: list[PlayerAction]) -> list[PlayerAction]:
    """Removes actions that are expected to always be bad choices."""
    filtered = list(itertools.filterfalse(_is_stupid, actions))
    if filtered:
        return filtered
    else:
        # Getting here usually means that there are fossil takeover assets that nobody has enough money to scrap,
        # but someone has enough money to scrap their renewable/battery assets. The game is headed for a loss due
        # to unowned takeover assets, so we'll just return the stupid scrap actions to move it along.
        return actions


def play(cl: client.Client, num_players:int, less_stupid:bool, fetch_log:bool)->tuple[Game, str]:
    """Play a single game. Thread safe.
    
    Args:
        cl: API Client
        num_players: Number of players at the start of the game
        less_stupid: Whether to filter out bad action choices
        fetch_log: Whether to retrieve the full game log at the end.

    Returns: A tuple of (Game, log)
    """
    with contextlib.closing(GameClient(cl, num_players=num_players)) as g:
        # Choose actions randomly until there are none left (game is over)
        while g.possible_actions:
            if less_stupid:
                g.send_action(random.choice(filter_stupid_actions(g.possible_actions)))
            else:
                g.send_action(random.choice(g.possible_actions))
        
        # Get game data at the end
        if fetch_log:
            log = g.get_log()
        else:
            log = ""

        return (g.game, log)


def mask_stupid_actions(original_mask: np.ndarray):
    """Masks actions that are expected to always be bad choices."""
    ok_actions = np.array([1, 1, 0, 0, 0, 1, 1, 1, 1, 0, 0, 1, 1, 1, 1], dtype=np.int8)
    filtered = original_mask*ok_actions
    if np.array_equal(filtered, np.zeros(filtered.shape)):
        return original_mask
    else:
        return filtered


def playEnv(cl: client.Client, num_players:int, less_stupid:bool, fetch_log:bool)->tuple[Game, str]:
    """Play a single game using the PettingZoo environment.
    
    Args:
        cl: API Client
        num_players: Number of players at the start of the game
        fetch_log: Whether to retrieve the full game log at the end.

    Returns: A tuple of (Game, log)
    """
    with contextlib.closing(joulequest_env.JoulequestEnv(num_players, cl)) as env:
        env.reset()
        for agent in env.agent_iter():
            observation, _, termination, truncation, _ = env.last()

            if termination or truncation:
                action = None
            else:
                if isinstance(observation, dict) and "action_mask" in observation:
                    mask = observation["action_mask"]
                    if less_stupid:
                        mask = mask_stupid_actions(mask)
                else:
                    mask = None
                action = env.action_space(agent).sample(mask) # this is where you would insert your policy

            env.step(action)

        # Get game data at the end
        if fetch_log:
            log = env.get_log()
        else:
            log = ""

        return (env.game, log)


def main():
    parser = argparse.ArgumentParser(
                    prog='JouleQuest server runner',
                    description='Runs the joulequest server in a child process and communicates with it over a unix socket')
    parser.add_argument('--executable', required=True, help="Path to the rest_api executable")
    parser.add_argument('--games', default=1, type=int, help='Number of games to simulate')
    parser.add_argument('--num_players', default=4, type=int, help='Number of players per game')
    parser.add_argument('--verbose', default=False, action='store_true', help='Whether to print the full game status and log after each game')
    parser.add_argument('--suppress_child_output', default=False, action='store_true', help='Whether to suppress the stdout and stderr of the child process')
    parser.add_argument('--less_stupid', default=False, action='store_true', help='Whether to filter out objectively stupid game choices')
    parser.add_argument('--use_env', default=False, action='store_true', help='Whether to use the PettingZoo env.')
    parser.add_argument('--parallel', default=1, type=int, help='How many workers in the thread pool')
    args = parser.parse_args()

    fs: list[futures.Future] = []
    outcomes: collections.Counter[tuple[GameStatus, GameReason]] = collections.Counter()
    with ServerClient(args.executable, socket_path=UDS_PATH, suppress_output=args.suppress_child_output) as cl:
        with futures.ThreadPoolExecutor(max_workers=args.parallel) as tpe:
            # Send game playing tasks to the worker threads
            for _ in range(args.games):
                if args.use_env:
                    fs.append(tpe.submit(playEnv, cl=cl, num_players=args.num_players, less_stupid=args.less_stupid, fetch_log=args.verbose))
                else:
                    fs.append(tpe.submit(play, cl=cl, num_players=args.num_players, less_stupid=args.less_stupid, fetch_log=args.verbose))
            
            # Record summary stats and maybe print the outcome of each completed game
            for f in futures.as_completed(fs):
                (game, log) = f.result()
                outcomes[(game.status, game.reason)] += 1
                if args.verbose:
                    print(log)
                    print(game)
                    print("-"*80)

        # Print summary stats once all games are completed.
        for s, c in outcomes.most_common():
            (status, reason) = s
            print(f"count: {c}\t status: {status}\t reason: {reason}")



if __name__ == "__main__":
    main()
