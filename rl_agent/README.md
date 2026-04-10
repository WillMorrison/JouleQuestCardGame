To regenerate the client, run `uvx openapi-python-client generate --path ../src/cmd/rest_api/openapi.json --config ./openapi_python_client_config.yaml --output-path ./apiclient` from this directory.

To play games with random action choices, first build the `rest_api` server Go binary, then use it with `main.py`.

```sh
pushd ../src
go build ./cmd/rest_api
popd
uv run main.py --executable ../src/rest_api --less_stupid --games 100
```

To run the API test for the joulequest PettingZoo environment

```sh
uv run -m joulequest_env.env_test --executable ../src/rest_api
```

To train a policy model

```sh
uv run train.py --executable ../src/rest_api --tensorboard_dir log

uv run tensorboard --logdir 'log'
```