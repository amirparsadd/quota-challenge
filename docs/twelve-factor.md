# Twelve-factor app compatibility

This document maps [The Twelve-Factor App](https://12factor.net/) methodology to **quota-challenge**: how each factor is implemented today, which environment variables apply, and deliberate gaps or production-only differences. Related: [deployment.md](deployment.md), [design.md](design.md), [assumptions.md](assumptions.md).

## Summary

| Factor | Status | Notes |
|--------|--------|--------|
| I. Codebase | **Aligned** | One repo; two deployable binaries (`quota-gateway`, `region-quota-service`). |
| II. Dependencies | **Aligned** | Go modules; pinned tool/image versions in `versions.env`; distroless runtimes. |
| III. Config | **Aligned** | Runtime config in `cmd/*` from env (and Cobra flags that default from env). No prod config files. |
| IV. Backing services | **Aligned** | Postgres and OTLP are attached resources via URLs and standard OTel env. |
| V. Build, release, run | **Mostly aligned** | Multi-stage Docker build; run = container entrypoint + subcommand. Compose uses dev-only serve flags. |
| VI. Processes | **Aligned** | Gateway is stateless; region app holds no durable local state (Postgres is the system of record). |
| VII. Port binding | **Aligned** | Services bind ports from config; gateway prefers `PORT` (Heroku-style). |
| VIII. Concurrency | **Aligned** | Scale out by running more processes; shared nothing except backing stores. |
| IX. Disposability | **Aligned** | `SIGINT`/`SIGTERM` → graceful HTTP/gRPC shutdown; health probes in Compose. |
| X. Dev/prod parity | **Partial** | Compose mirrors topology; some compose-only behavior (`--auto-migrate`, `--seed-demo`, `--dev-reaper`). K8s/Helm not in repo yet. |
| XI. Logs | **Aligned** | Structured JSON on **stdout** via `log/slog`; level from `LOG_LEVEL`. |
| XII. Admin processes | **Aligned** | Migrations, reaper, drift check run as one-off CLI subcommands, not in-container shells. |

---

## I. Codebase — one codebase tracked in revision control, many deploys

**Principle:** One app per repo (or one coherent product); many deploys from the same revision.

**Here:**

- Single Git repository for the gateway, region service, contracts (`api/openapi`, `proto/`), migrations, Compose, and CI.
- **Two binaries**, not two repos: `cmd/quota-gateway` and `cmd/region-quota-service`. Production deploys one or more instances of each (plus Postgres per region).
- Generated code under `gen/` is committed when protos change so builds are reproducible from a single revision.

**Deploy variants:** default Compose (one region), `docker-compose.multiregion.yml` (fan-out e2e), and the documented future target (Kubernetes per [deployment.md](deployment.md)) all build from this same tree.

---

## II. Dependencies — explicitly declare and isolate dependencies

**Principle:** Do not rely on implicit system packages; declare and lock dependencies.

**Here:**

| Layer | Mechanism |
|-------|-----------|
| Go | `go.mod` / `go.sum`; `CGO_ENABLED=0` static binaries |
| Protobuf / API | `buf` (`buf.lock`); OpenAPI lint via Spectral in `make lint` |
| Containers | Base images pinned in `versions.env` and `Dockerfile` (`golang:…`, `postgres:…`, distroless mirror in `Dockerfile`) |
| Runtime image | Distroless `static-debian12:nonroot` — no shell, no package manager in the run image |

Application code does not read arbitrary files from the image filesystem for business logic (migrations are bundled in the region image at `/migrations` for the `migrate` subcommand only).

---

## III. Config — store config in the environment

**Principle:** Config that varies between deploys belongs in the environment, not in the codebase.

**Here:** All process-level configuration is resolved in `cmd/quota-gateway` and `cmd/region-quota-service` from environment variables and Cobra flags (flags default from env where applicable). There are **no** production config files checked into the repo for services.

### Gateway (`quota-api-gateway`)

| Variable | Purpose | Default / notes |
|----------|---------|-----------------|
| `PORT` | HTTP listen (12-factor) | If set, binds `:<PORT>` |
| `HTTP_ADDR` | HTTP listen | `:8080` if `PORT` unset |
| `REGIONS` | Region registry | `id=host:port,...` e.g. `ir-thr-1=region-quota-service:9090` |
| `LOG_LEVEL` | slog level | `info` |
| `OTEL_SDK_DISABLED` | Disable telemetry | unset = enabled |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OTLP HTTP endpoint | unset = stdout exporters |
| `OTEL_SERVICE_NAME` | Service name in telemetry | set in Compose |

Listen address resolution explicitly prefers `PORT`:

```15:24:cmd/quota-gateway/config.go
// listenHTTPAddr prefers PORT (12-factor), then HTTP_ADDR.
func listenHTTPAddr() string {
	if p := os.Getenv("PORT"); p != "" {
		if strings.HasPrefix(p, ":") {
			return p
		}
		return ":" + p
	}
	return env("HTTP_ADDR", ":8080")
}
```

### Region service (`region-quota-service`)

| Variable | Purpose | Required for `serve` |
|----------|---------|----------------------|
| `REGION_ID` | Must match `region_meta` | Yes |
| `DATABASE_URL` | Postgres DSN | Yes |
| `GRPC_ADDR` | gRPC bind | Flag default `:9090` |
| `DB_POOL_MAX_CONNS` | pgxpool max connections (`serve` only) | No (pgx default) |
| `DB_POOL_MIN_CONNS` | pgxpool minimum idle connections | No |
| `DB_POOL_MAX_CONN_LIFETIME` | Max connection lifetime (Go duration, e.g. `30m`) | No |
| `DB_POOL_MAX_CONN_IDLE_TIME` | Max time a connection may stay idle before recycle | No |
| `LOG_LEVEL`, `OTEL_*` | Same pattern as gateway | No |

**Secrets:** Compose uses `.env` (from `.env.example`, gitignored) for Postgres credentials and local overrides. Production target: Vault → External Secrets → Kubernetes Secret ([deployment.md](deployment.md)) — not implemented in the submission artifact.

**Not config (code):** Domain rules, OpenAPI shapes, SQL migrations, and default demo seed **data** live in the repo; only connection strings, region IDs, region routing, and operational tuning belong in env.

---

## IV. Backing services — treat backing services as attached resources

**Principle:** Databases, queues, and caches are resources reachable via URL/credentials in config; swap providers without code changes.

**Here:**

| Resource | Attachment | Swap example |
|----------|------------|----------------|
| **PostgreSQL** | `DATABASE_URL` | Local Compose Postgres → CNPG cluster; change DSN only |
| **OTLP collector** | `OTEL_EXPORTER_OTLP_ENDPOINT` | stdout in dev → centralized collector in prod |
| **Region gRPC peers** | `REGIONS` on gateway | Add/remove region backends by env |

The gateway does not embed Postgres; each region service owns one database. No Redis or other mandatory backing service in scope ([design.md](design.md)).

---

## V. Build, release, run — strictly separate build and run stages

**Principle:** Build produces artifacts; release combines build + config; run executes processes.

**Here:**

| Stage | What happens |
|-------|----------------|
| **Build** | `Dockerfile` compile stage: `go build` → `/out/quota-gateway`, `/out/region-quota-service` |
| **Release** | Image tags (local `make up`, future CI/registry in [future-work.md](future-work.md)); GoReleaser workflow on version tags for binaries |
| **Run** | Distroless image `ENTRYPOINT` + subcommand: default `serve` for long-running; `migrate`, `reap`, `drift-check`, `healthcheck` for jobs/probes |

Compose **run** overrides region `command` with dev helpers:

- `--auto-migrate` — apply migrations at startup (documented **Compose-only**; production should use a migrate Job).
- `--seed-demo` — load demo catalog/quota (**dev only**).
- `--dev-reaper=5s` — in-process TTL reaper (**dev only**; production uses CronJob `reap`).

These flags violate strict dev/prod parity on purpose for a one-command local stack; they must not appear in production rollouts ([deployment.md](deployment.md)).

---

## VI. Processes — execute the app as one or more stateless processes

**Principle:** Durable state goes to backing services; processes are interchangeable.

**Here:**

- **Gateway:** No session store, no local quota data. In-memory circuit breakers and gRPC client pools are rebuildable on restart; idempotency and quotas live in region databases ([design.md](design.md)).
- **Region service:** Business state in PostgreSQL (ledger, counters, idempotency keys). Process restart loses only in-flight requests, not committed data.
- **Horizontal scale:** Multiple gateway replicas behind a load balancer; multiple region replicas against one primary Postgres (reads may use replicas in production per [scaling.md](scaling.md)).

---

## VII. Port binding — export services via port binding

**Principle:** The app is self-contained and binds to a port; no requirement to run under a specific web server in the container.

**Here:**

- Gateway: HTTP server in-process (`net/http`), address from `PORT` / `HTTP_ADDR`.
- Region: gRPC server in-process, address from `GRPC_ADDR` (default `:9090`).
- Compose publishes only the gateway to the host (`HTTP_BIND`:`HTTP_PORT`); region and Postgres stay on internal networks.

---

## VIII. Concurrency — scale out via the process model

**Principle:** Scale by running more processes; avoid sticky in-process assumptions where possible.

**Here:**

- Go’s per-request goroutine model handles concurrent HTTP/gRPC within one process.
- **Scale-out** = more container/Pod replicas. Gateway fan-out is parallel per region; region DB is the write bottleneck per region ([scaling.md](scaling.md)).
- No requirement for sticky sessions on the gateway. Idempotency is keyed by client `Idempotency-Key` and stored in Postgres, safe across gateway instances.

---

## IX. Disposability — maximize robustness with fast startup and graceful shutdown

**Principle:** Processes start quickly, shut down gracefully on `SIGTERM`, and tolerate sudden death when the platform requires it.

**Here:**

| Behavior | Implementation |
|----------|------------------|
| **Startup** | Open pool, optional migrate/seed (dev), dial regions, listen |
| **Shutdown** | Gateway: 10s bounded `http.Server.Shutdown` on signal; region: `grpc.Server.GracefulStop` |
| **Health** | gRPC health service (region); `healthcheck` subcommand for Compose; gateway relies on process up + backend dial at start |
| **Compose** | `stop_grace_period: 30s` on app containers |

Telemetry providers flush on deferred shutdown hooks after signal handling.

---

## X. Dev/prod parity — keep development, staging, and production as similar as possible

**Principle:** Time gaps, tooling gaps, and content gaps between dev and prod should be small.

**Aligned aspects:**

- Same binaries and images (distroless) locally and in CI e2e.
- Same env var names for config (`DATABASE_URL`, `REGIONS`, `LOG_LEVEL`, `OTEL_*`).
- Same data model and migrations; integration tests use real Postgres (testcontainers).

**Known gaps (documented):**

| Gap | Dev (Compose) | Production target |
|-----|---------------|---------------------|
| Migrations | `--auto-migrate` on `serve` | One-shot `migrate` Job before rollout |
| Demo data | `--seed-demo` | Seeding via admin tooling / GitOps |
| Reservation expiry | `--dev-reaper` in process | `reap` CronJob each minute |
| Secrets | `.env` file | Vault + ESO |
| Orchestration | Docker Compose | Kubernetes + Helm (**not in repo yet**) |
| Auth | `X-Customer-ID` / `x-caller` stubs | JWT/API keys, mTLS |

See [future-work.md](future-work.md) for planned closure of deploy/tooling gaps without changing submission scope.

---

## XI. Logs — treat logs as event streams

**Principle:** Apps never manage log files; emit unbuffered streams to stdout for the platform to collect.

**Here:**

- `internal/obslog.SetupDefault` configures **JSON** `slog` handlers writing to **stdout**.
- `LOG_LEVEL` controls verbosity (`debug` enables HTTP/gRPC/SQL detail).
- HTTP 4xx → warn, 5xx → error; successful requests often debug ([design.md](design.md) observability).
- No log rotation or file paths in application code.

In Compose, the Docker logging driver captures stdout; in Kubernetes, the container runtime forwards the same stream to your log stack.

---

## XII. Admin processes — run admin/management tasks as one-off processes

**Principle:** Do not SSH into running containers for schema changes or batch jobs; use the same release artifact with a different command.

**Here (`region-quota-service`):**

| Subcommand | Role |
|------------|------|
| `migrate` | Apply goose migrations (`DATABASE_URL`) |
| `reap` | Expire reservations (batch) |
| `drift-check` | Dry-run counter resync / drift detection |
| `healthcheck` | Exit 0 if DB reachable (Compose probe) |

Long-running **`serve`** is separate from these admin paths. Distroless images have no shell — admin work must use the binary subcommands or an init/job container with the same image.

Gateway has no DB migrations; its operational surface is config-only deploys and scaling replicas.

---

## Checklist for new deploy environments

When wiring a new environment (staging, second region, future K8s):

1. Set **`DATABASE_URL`**, **`REGION_ID`**, and gateway **`REGIONS`** from the environment or secret store — not from files in the image.
2. Run **`migrate`** (or a migrate Job) before pointing traffic at a new region revision.
3. Schedule **`reap`** and **`drift-check`** outside the `serve` process.
4. Point **`OTEL_EXPORTER_OTLP_ENDPOINT`** at your collector (or disable with `OTEL_SDK_DISABLED=true`).
5. Avoid compose-only flags: **`--auto-migrate`**, **`--seed-demo`**, **`--dev-reaper`**.
6. Ensure **`PORT`** or **`HTTP_ADDR`** / **`GRPC_ADDR`** match platform port binding and health checks.

For local development, `cp .env.example .env` and `make up` remain the reference dev/prod-like stack ([quickstart.md](quickstart.md)).