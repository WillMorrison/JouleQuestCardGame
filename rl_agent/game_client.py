import contextlib
import subprocess

import httpx

from apiclient import joule_quest_api_client as client
from apiclient.joule_quest_api_client.api.default import post_new, delete_g_game_id, post_g_game_id_action, get_g_game_id_log
from apiclient.joule_quest_api_client.models import GameUpdate, Error, PlayerAction, Game

@contextlib.contextmanager
def ServerClient(executable: str, socket_path: str, suppress_output: bool=False):
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
