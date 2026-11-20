# Agent guide

Instructions for automated agents working in **quota-challenge** (multi-region quota gateway + gRPC region services).

## What this system does

- **`quota-api-gateway`** — public REST (`:8080` / `PORT`); ops HTTP (`OPS_ADDR` default `:8081`) for `/livez`, `/readyz`, `/metrics`; fans out to regions.
- **`region-quota-service`** — gRPC per region; Postgres ledger, TTL reservations, idempotency.

Binary layout and packages: [docs/structure.md](docs/structure.md). Architecture: [docs/design.md](docs/design.md).

## Repository map

| Area | Purpose |
|------|---------|
| `cmd/quota-gateway` | REST gateway CLI (`serve` default) |
| `cmd/region-quota-service` | Region CLI: `serve`, `migrate`, `reap`, `drift-check`, `healthcheck` |
| `internal/gateway` | HTTP handlers, problem+json, OpenAPI `/docs` |
| `internal/region/regionsvc` | Domain logic (reserve, confirm, cancel, …) |
| `internal/region/postgres` | SQL store + goose migrations in `migrations/` |
| `internal/region/grpcserver` | gRPC mapping |
| `api/openapi`, `api/proto` | Contracts; run `buf generate` after proto changes |
| `api/gen/` | Generated Go from buf (commit when proto changes) |
| `test/integration/` | `//go:build integration` + testcontainers Postgres |
| `test/e2e/` | `//go:build e2e` + multiregion Compose |
| `deployments/` | `docker-compose.yml`, multiregion override |
| `build/` | Per-binary Dockerfiles |
| `configs/` | otel/prometheus configs for Compose |
| `versions.env` | Pinned toolchain and image versions (root; Makefile, CI) |

## How to develop

1. **Config** — env-only in `cmd/*` (12-factor). No config files in prod paths.
   - Gateway: `PORT` or `HTTP_ADDR`, `REGIONS` (`id=host:port,…`).
   - Region: `DATABASE_URL` (required for `serve`), `REGION_ID`, `GRPC_ADDR`, OTLP via `OTEL_*`.
2. **Pins** — image/tool versions in `versions.env`; compose overrides via `.env` (from `.env.example`).
3. **Tests** — `make test` (unit + integration); `make test-e2e` (multiregion). Do not rely on `go test ./...` alone (build tags).
4. **Lint** — `make lint` (golangci-lint if installed, buf, spectral, govulncheck).
5. **Migrations** — SQL in `internal/region/postgres/migrations/`; apply via `region-quota-service migrate` or `serve --auto-migrate` (compose only).
6. **Commits** — [Conventional Commits](https://www.conventionalcommits.org/); validated by gitlint (`make lint-commits`, `commit-msg` hook via `make install-hooks`, or CI). Install: `pip install gitlint==0.19.1` (pin in `versions.env`). Use **SSH-signed** commits (`git commit -S`; signing enabled in this repo).
7. **Scope** — match existing style: small files by concern, pgx without ORM, `log/slog` to stdout, no unrelated refactors.

## Common tasks

| Task | Command / location |
|------|-------------------|
| Git commit-msg hook | `make install-hooks` (needs `gitlint` on `PATH`) |
| Lint commits on branch | `make lint-commits` or `scripts/lint-commits.sh [range]` |
| Local stack | `cp .env.example .env` → `make up` (Compose: `deployments/`) |
| Multiregion | `make up-multiregion` |
| Proto change | edit `api/proto/` → `buf generate` → commit `api/gen/` |
| OpenAPI change | edit `api/openapi/v1.yaml` → spectral via `make lint` |
| Release binaries | tag `v*` → GoReleaser workflow (`.goreleaser.yaml`) |

## CI workflows

| Workflow | Role |
|----------|------|
| `test.yml` | Lint, unit/integration, e2e Compose |
| `commits.yml` | gitlint on PR commit messages |
| `security.yml` | CodeQL, govulncheck, gosec, Trivy FS |
| `release.yml` | GoReleaser on version tags |

## Do not

- Commit `.env` or secrets.
- Add `latest` image/tool tags — use `versions.env`.
- Run production-only flags in compose (`--seed-demo`, `--dev-reaper`) outside dev docs.

## References

- [docs/deployment.md](docs/deployment.md) — Compose, networks, volumes, distroless mirror.
- [docs/assumptions.md](docs/assumptions.md) — submission scope.
- [docs/future-work.md](docs/future-work.md) — intentional gaps (Helm, mTLS, etc.).