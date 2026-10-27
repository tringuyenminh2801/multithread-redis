# multithread-redis

A Redis-compatible server in Go, built around a fixed goroutine pool.

## Layout

```
cmd/server/            entrypoint: config, logging, metrics, signal handling
internal/config/       defaults < env vars < flags
internal/pool/         goroutine pool that handles connections
internal/protocol/     RESP encoding/decoding
internal/server/       TCP accept loop
internal/observability/ JSON logging (slog), /metrics, /healthz
build/Dockerfile       multi-stage, distroless image
deploy/app/            app host stack: redis + Alloy + cAdvisor + node-exporter
deploy/monitoring/     monitoring stack: Grafana + Loki + Mimir
.github/workflows/     CI, app deploy, monitoring deploy
```

## Configuration

| Flag | Env var | Default |
|---|---|---|
| `-port` | `REDIS_PORT` | `:6379` |
| `-pool-size` | `REDIS_POOL_SIZE` | `100000` |
| `-max-waiting-conns` | `REDIS_MAX_WAITING_CONNS` | `10000` |
| `-buffer-size` | `REDIS_BUFFER_SIZE` | `2048` |
| `-metrics-addr` | `REDIS_METRICS_ADDR` | `:2112` |
| `-log-level` | `REDIS_LOG_LEVEL` | `info` |

Flags override env vars, env vars override defaults.

## Run locally

```sh
go run ./cmd/server -port :7000
go test ./...
```

## Observability

```
app host                                          monitoring host (may be the same)
  redis :2112/metrics ─┐
  cAdvisor ────────────┼─► Alloy ──metrics──► Mimir :9009 ─┐
  node-exporter ───────┘     │                              ├─► Grafana :3000
  container stdout ──────────┴────logs──────► Loki  :3100 ─┘
```

Run both stacks locally (needs Docker):

```sh
cp deploy/monitoring/.env.example deploy/monitoring/.env
cp deploy/app/.env.example deploy/app/.env
make monitoring-up      # Grafana: http://localhost:3000
make app-up             # builds and runs the server + collectors
```

The app stack finds the monitoring stack through `LOKI_URL` and `MIMIR_URL`.
Moving monitoring to its own host only means changing those two values.
Loki and Mimir have no auth, so on a separate host only expose ports 3100/9009 to app hosts.

## Deploy (GitHub Actions)

Create two GitHub Environments under **Settings → Environments**:

**`production`** (app host)
- secrets: `SSH_HOST`, `SSH_USER`, `SSH_PRIVATE_KEY`
- variables: `REDIS_POOL_SIZE`, `REDIS_MAX_WAITING_CONNS`, `REDIS_BUFFER_SIZE`, `REDIS_LOG_LEVEL`, `LOKI_URL`, `MIMIR_URL`, `HOST_NAME`

**`monitoring`** (monitoring host, can be the same machine)
- secrets: `SSH_HOST`, `SSH_USER`, `SSH_PRIVATE_KEY`, `GRAFANA_ADMIN_PASSWORD`

Hosts need Docker with the compose plugin.
`deploy.yml` builds the image, pushes it to GHCR, and runs `docker compose up` on the app host.
`deploy-monitoring.yml` runs when `deploy/monitoring/**` changes, or manually.
