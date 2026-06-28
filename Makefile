BUILD_DIR := build

REST_API := $(BUILD_DIR)/rest_api
WASM := $(BUILD_DIR)/joulequest.wasm

GO := go
TINYGO := tinygo
UV := uv

.PHONY: rest-api wasm clean test format go-generate gen-rest-api-client gen-wasm-api-client check-build-tools

# Build artifacts

$(BUILD_DIR):
	mkdir -p $@

rest-api: | $(BUILD_DIR)
	$(GO) -C src build -o ../$(REST_API) ./cmd/rest_api

wasm: | $(BUILD_DIR)
	$(TINYGO) build -C src -size short -gc=none -no-debug -scheduler=none -panic=trap \
		-target=wasm-unknown -o ../$(WASM) ./compact/wasm

clean:
	rm -rf $(BUILD_DIR)

# Dev workflow

test:
	$(GO) -C src test ./...

format:
	$(GO) -C src fmt ./...
	$(UV) tool run --directory rl_agent/ ruff format .

# Code generation

go-generate:
	$(GO) -C src generate ./...

gen-rest-api-client:
	rm -rf rl_agent/apiclient
	$(UV) tool run --directory rl_agent openapi-python-client generate \
		--path ../src/cmd/rest_api/openapi.json \
		--config ./openapi_python_client_config.yaml \
		--output-path ./apiclient
	$(UV) tool run --directory rl_agent/ ruff format ./apiclient

gen-wasm-api-client:
	$(GO) -C src run ./cmd/wasm_pybindgen/ --out_dir '../rl_agent/wasm_api_client/'
	$(UV) tool run --directory rl_agent/ ruff format './wasm_api_client'

# Check dependencies

check-build-tools:
	@command -v $(GO) >/dev/null 2>&1 || { echo "missing: $(GO)"; exit 1; }
	@command -v $(TINYGO) >/dev/null 2>&1 || { echo "missing: $(TINYGO)"; exit 1; }
	@command -v $(UV) >/dev/null 2>&1 || { echo "missing: $(UV)"; exit 1; }
	@echo "ok: $(GO) ($$($(GO) version))"
	@echo "ok: $(TINYGO) ($$($(TINYGO) version))"
	@echo "ok: $(UV) ($$($(UV) --version))"
