# Deployment

## Submission: Docker Compose

```bash
cp .env.example .env   # once: Postgres password, HTTP bind/port, OTEL
# Image/tool pins live in versions.env (Postgres, Go, buf, Spectral).
make up          # build + run (default: one region)
make down
make test
make lint        # go test, golangci-lint, buf, openapi check — see design.md
make test-e2e    # docker-compose.multiregion.yml when present
```

**Compose-only flags on region service:** `--auto-migrate`, `--seed-demo`, `--dev-reaper=5s` — never use in production.

| Service | Image target | Ports | Notes |
|---|---|---|---|
| `quota-api-gateway` | `quota-gateway` | 8080 public API; 8081 ops (internal) | `REGIONS=ir-thr-1=region-quota-service:9090` |
| `otel-collector` | contrib collector | internal 4318 OTLP | forwards metrics to Prometheus exporter |
| `prometheus` | `prom/prometheus` | `${METRICS_BIND}:9091` → UI | scrapes gateway `/metrics` and collector |
| `region-quota-service` | `region-quota-service` | 9090 localhost | `REGION_ID`, `DATABASE_URL` |
| `postgres` | `postgres:16.16-bookworm` (see `versions.env`) | internal only (`quota-internal`) | named volume `postgres-data`; schema via migrate / `--auto-migrate` |

**Quickstart (curl flows):** [quickstart.md](quickstart.md)

**API docs:** with the gateway up, open `http://localhost:8080/docs` (Scalar UI over embedded `api/openapi/v1.yaml`).

**Try the API**

| Need | Call |
|---|---|
| Products you can create | `GET /v1/regions/ir-thr-1/products` |
| Your quotas (seeded customer) | `GET /v1/regions/ir-thr-1/quota` with `X-Customer-ID: demo` |
| Error code meanings | `GET /v1/problems` |

Example create:

```bash
curl -s -X POST localhost:8080/v1/regions/ir-thr-1/services \
  -H 'X-Customer-ID: demo' -H 'Idempotency-Key: 11111111-1111-1111-1111-111111111111' \
  -H 'Content-Type: application/json' \
  -d '{"product":"vm","items":[{"resource":"cpu","amount":2},{"resource":"ram","amount":4},{"resource":"disk","amount":40},{"resource":"public_ip","amount":1}]}'
```

**Logging:** JSON lines on stdout via `log/slog`. Set `LOG_LEVEL=debug` for HTTP/gRPC/SQL detail. Level defaults to `info` (startup and errors). Business **4xx** HTTP responses log at **warn**; **5xx** at **error**.

**OpenTelemetry:** Compose sets `OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4318`; the collector exposes a Prometheus scrape target and apps also expose OTLP-derived metrics at `GET :8081/metrics` on the gateway ops port. Without Compose, metrics/traces go to stdout unless OTLP is set. Disable with `OTEL_SDK_DISABLED=true`.

**Twelve-factor config:** processes are stateless; logs go to stdout (`log/slog`); config via environment (`DATABASE_URL`, `GRPC_ADDR`, `PORT`/`HTTP_ADDR`, `REGIONS`, `OTEL_*`). Admin tasks use CLI subcommands (`migrate`, `reap`), not shells in the image.

**Compose env & networking:** copy [`.env.example`](../.env.example) to `.env` (gitignored). Postgres credentials, HTTP bind/port, `LOG_LEVEL`, and OTLP are substituted from `.env`. App traffic uses internal network `quota-internal`; the gateway also attaches to `quota-edge` and publishes `${HTTP_BIND}:${HTTP_PORT}` (default `127.0.0.1:8080`). Postgres data lives in named volumes (`postgres-data`, or `postgres-a-data` … in multiregion). `make down` passes `-v` and deletes those volumes.

**Multiregion:** override adds two more region DBs + services for isolation and fan-out e2e tests.

**Images:** multi-stage `Dockerfile` → **distroless** `static-debian12:nonroot` runtimes via Shecan mirror `mirror-gcr.pkg.shecan.ir/distroless/static-debian12:nonroot` (hardcoded; same path as `gcr.io/distroless/...`). Subcommands: gateway `serve`, `healthcheck`; region `serve`, `migrate`, `reap`, `drift-check`, `healthcheck`.

## Production reference (not in submission)

Documented target environment — **not shipped in this repo yet** (planned later; see [future-work.md](future-work.md)):

- **Kubernetes + Helm** (`deploy/` — **directory not present**; to be added): gateway (≥2), region service per region (≥3), CNPG per region, CronJobs `reap` / `drift-check`, migrate Job pre-rollout.
- **Secrets:** Vault → ESO → K8s Secret (compose: `.env` / defaults in `.env.example` for local dev only).
- **Auth:** API key/JWT at gateway; mTLS on internal gRPC.
- **Delivery:** GitHub Actions image build, signing, optional container scan on `main`, GitOps to clusters — **later**; today CI only runs tests and e2e.

See [design.md](design.md) for health probes, graceful shutdown, and failure behavior.