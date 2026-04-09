import argparse
from typing import Final

from pettingzoo.test import api_test as aec_api_test

import joulequest_env
from game_client import ServerClient

UDS_PATH: Final[str] = "/tmp/joulequest_api_env_test.sock"

def main():
    parser = argparse.ArgumentParser(
                    prog='JouleQuest server runner',
                    description='Runs the joulequest server in a child process and communicates with it over a unix socket')
    parser.add_argument('--executable', required=True, help="Path to the rest_api executable")
    args = parser.parse_args()

    with ServerClient(args.executable, socket_path=UDS_PATH, suppress_output=True) as cl:
        aec_api_test(joulequest_env.env(num_players=4, client=cl))


if __name__ == "__main__":
    main()