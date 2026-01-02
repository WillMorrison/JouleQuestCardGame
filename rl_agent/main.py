import argparse
import collections
from concurrent import futures
import contextlib
import itertools
import random
import subprocess
from typing import Final

import httpx

from apiclient import joule_quest_api_client as client
from apiclient.joule_quest_api_client.api.default import post_new, delete_g_game_id, post_g_game_id_action, get_g_game_id_log
from apiclient.joule_quest_api_client.models import GameUpdate, Error, PlayerAction, Game, GameReason, GameStatus, PlayerActionType, PlayerActionAssetType
from apiclient.joule_quest_api_client.types import Unset


UDS_PATH: Final[str] = "/tmp/joulequest_api.sock"


@contextlib.contextmanager
def ServerClient(executable: str, socket_path: str = UDS_PATH, suppress_output: bool=False):
    """
    Manages a rest_api child process and the Unix socket used to communicate with it.
    
    Args:
      executable: The path to the rest_api binary to start in a child process.
      socket_path: The path to create a unix socket at for communicating with the child process.
    """
    # Spawn server as child process
    child_process = subprocess.Popen(
        [executable, '--socket', socket_path],
        stderr=subprocess.DEVNULL if suppress_output else None,
        stdout=subprocess.DEVNULL if suppress_output else None,
    )

    # Set up and return an API client
    transport = httpx.HTTPTransport(uds=socket_path)
    try:
        with client.Client(
            base_url='http://socket',
            httpx_args=dict(transport=transport),
            verify_ssl=False,
            timeout=httpx.Timeout(2.0),
            raise_on_unexpected_status=True,
        ) as cl:
            yield cl
    finally:
        # Send the child process the terminate signal and wait for it to disappear.
        child_process.terminate()
        child_process.wait()


class GameError(Exception):
    """Raised on Error responses from the API server."""


class GameClient:
    def __init__(self, client: client.Client, num_players: int):
        self._client = client
        self._last_update: GameUpdate | None = None
        self._active = False

        self._new_game(num_players=num_players)
        
    @property
    def id(self) -> str:
        if self._last_update:
            return self._last_update.id
        else:
            return ""

    @property
    def possible_actions(self) -> list[PlayerAction]:
        if not self._last_update or not self._last_update.possible_actions:
            return []
        return self._last_update.possible_actions
    
    @property
    def game(self) -> Game:
        if not self._last_update:
            raise GameError("Not Initialized")
        return self._last_update.game

    def _new_game(self, num_players: int) -> None:
        r = post_new.sync(client=self._client, num_players=num_players)
        if isinstance(r, Error):
            raise GameError(r.error)
        elif r is None:
            raise GameError("Unknown response")

        self._last_update = r
        self._active = True

    def _delete_game(self) -> None:
        if not self._active or not self.id:
            return
        r = delete_g_game_id.sync(client=self._client, game_id=self.id)
        if isinstance(r, Error):
            raise GameError(r.error)

    def close(self):
        self._delete_game()
        self._active = False
    
    def send_action(self, action: PlayerAction) -> None:
        if not self._active or not self.id:
            return
        
        r = post_g_game_id_action.sync(self.id, client=self._client, body=action)
        if isinstance(r, Error):
            raise GameError(r.error)
        elif r is None:
            raise GameError("Unknown response")

        self._last_update = r
        
    def get_log(self) -> str:
        if not self._active or not self.id:
            return ""
        
        raw_response = get_g_game_id_log.sync_detailed(self.id, client=self._client)
        r = raw_response.parsed
        if isinstance(r, Error):
            raise GameError(r.error)
        elif r is None:
            raise GameError("Unknown response")

        return raw_response.content.decode()

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
    args = parser.parse_args()

    fs: list[futures.Future] = []
    outcomes: collections.Counter[tuple[GameStatus, GameReason]] = collections.Counter()
    with ServerClient(args.executable, socket_path=UDS_PATH, suppress_output=args.suppress_child_output) as cl:
        with futures.ThreadPoolExecutor(max_workers=1) as tpe:
            # Send game playing tasks to the worker threads
            for _ in range(args.games):
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
