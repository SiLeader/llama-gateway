# llama-gateway

A lightweight gateway server that sits in front of [llama.cpp](https://github.com/ggml-org/llama.cpp)'s `llama-server`, providing automatic model downloading from Hugging Face and an OpenAI-compatible API.

## Features

- **Automatic model downloads** — fetches GGUF models from Hugging Face on startup, with SHA256 verification
- **llama-server lifecycle management** — spawns and supervises `llama-server`, restarting it automatically on crash
- **Full llama-server API passthrough** — all endpoints supported by `llama-server` are available as-is
- **Model name mapping** — maps user-friendly model names to the underlying GGUF files via `presets.ini`
- **Runtime model registration** — optional admin API to add models without restarting the gateway
- **Docker support** — includes a multi-stage Dockerfile and Docker Compose config

## Requirements

- Go 1.22+
- [llama.cpp](https://github.com/ggml-org/llama.cpp) `llama-server` binary (bundled in the Docker image)
- A Hugging Face account (token required for gated models)

## Configuration

Configuration is loaded from a YAML file (default: `/etc/llama-gateway/config.yaml`).

```yaml
server:
  listen:
    host: 0.0.0.0                           # default: 0.0.0.0
    port: 8080                              # default: 8080
  apis:
    addModels: true                         # enable POST /gateway/v1/models (default: false)
  auth:
    adminKeyEnv: LLAMA_GATEWAY_ADMIN_KEY    # env var holding the admin key (default shown)

models:
  - name: gemma3-270m                          # model name exposed in the API
    id: ggml-org/gemma-3-270m-it-qat-GGUF      # Hugging Face repo ID
    file: gemma-3-270m-it-qat-Q4_0.gguf        # filename inside the repo
    context: 4096                              # optional context size
  - name: ruri-v3-310m
    id: Targoyle/ruri-v3-310m-GGUF
    file: ruri-v3-310m-q8_0.gguf

directories:
  models: /var/run/llama-gateway/models   # where downloaded models are stored
  config: /var/run/llama-gateway/config   # where presets.ini is written

backend:
  port: 8081                              # llama-server port (default: server.listen.port + 1)
  llamaServer:
    executable: /opt/llama.cpp/llama-server
    args: ["--embeddings"]                # extra args forwarded to llama-server
    threads: 4                            # optional; defaults to detected CPU count (cgroup-aware)
```

### Authentication for the inference API

llama-gateway does **not** authenticate requests to the forwarded inference endpoints itself. Configure authentication on `llama-server` instead — either by adding its CLI flags to `backend.llamaServer.args` (e.g. `["--api-key", "…"]`) or by passing environment variables to the gateway process, which are inherited by the spawned `llama-server`. Because of this, any auth-related environment variables documented by `llama-server` should be set on the llama-gateway process itself.

### Environment variables

| Variable                   | Description                                                                       |
|----------------------------|-----------------------------------------------------------------------------------|
| `HF_TOKEN`                 | Hugging Face API token for downloading gated models                               |
| `LOG_LEVEL`                | Log verbosity: `debug`, `info` (default), `warn`, `error`                         |
| `LLAMA_GATEWAY_ADMIN_KEY`  | Admin key for the gateway admin API; name configurable via `server.auth.adminKeyEnv`. Required when `server.apis.addModels` is `true`. |

Any additional environment variables set on the llama-gateway process are inherited by the spawned `llama-server` (see above).

A `.env` file in the working directory is loaded automatically if present.

## Running

### With Docker Compose

```bash
docker compose up
```

The gateway listens on port `8080`. The `config/` directory is mounted at `/etc/llama-gateway`.

### From source

```bash
go build -o llama-gateway
./llama-gateway -config /path/to/config.yaml
```

### CLI flags

| Flag       | Default                          | Description        |
|------------|----------------------------------|--------------------|
| `-config`  | `/etc/llama-gateway/config.yaml` | Path to config file |

## API

The gateway transparently forwards all requests to `llama-server`, so every endpoint that `llama-server` supports is available. Refer to the [llama.cpp server documentation](https://github.com/ggml-org/llama.cpp/blob/master/tools/server/README.md) for the full API reference.

### Example: Responses

```bash
curl http://localhost:8080/v1/responses \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $OPENAI_API_KEY" \
  -d '{
    "model": "gemma3-270m",
    "input": "What is the capital of France?"
  }'
```

### Example: Embeddings

```bash
curl http://localhost:8080/v1/embeddings \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $OPENAI_API_KEY" \
  -d '{
    "model": "ruri-v3-310m",
    "input": "What is the capital of France?",
    "encoding_format": "float"
  }'
```

### Gateway admin API

When `server.apis.addModels` is `true`, the gateway exposes endpoints under `/gateway/` for managing models at runtime. These are handled by the gateway itself and are **not** forwarded to `llama-server`. They require the admin key in the `X-Llama-Gateway-Api-Key` header.

#### `POST /gateway/v1/models`

Adds a new model. The file is downloaded from Hugging Face, `presets.ini` is rewritten, and `llama-server` is restarted to pick up the new preset.

```bash
curl http://localhost:8080/gateway/v1/models \
  -H "Content-Type: application/json" \
  -H "X-Llama-Gateway-Api-Key: $LLAMA_GATEWAY_ADMIN_KEY" \
  -d '{
    "name": "gemma3-270m",
    "id": "ggml-org/gemma-3-270m-it-qat-GGUF",
    "file": "gemma-3-270m-it-qat-Q4_0.gguf",
    "context": 4096
  }'
```

Returns `201 Created` on success, `403 Forbidden` if the admin key is missing or incorrect, and `404 Not Found` if `addModels` is disabled.

## Architecture

```
Client
  │
  ▼
llama-gateway  (port 8080)
  │  - downloads models from Hugging Face on startup
  │  - resolves model name → GGUF file path
  │  - reverse-proxies requests
  ▼
llama-server   (port 8081, internal)
```

On startup the gateway:
1. Downloads all configured models from Hugging Face (skips files that already match the expected SHA256).
2. Writes a `presets.ini` file consumed by `llama-server`'s `--models-preset` flag.
3. Starts `llama-server` on `backend.port` (or `server.listen.port + 1` if unset) and supervises it, restarting on crash.
4. Starts an HTTP server that reverse-proxies incoming requests to `llama-server` and handles `/gateway/*` admin endpoints itself.

## License

MIT
