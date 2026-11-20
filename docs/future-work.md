# Future work (post-submission)

Items called out in review that are **intentionally out of scope** for the Compose deliverable but should land in follow-up work. The submission focuses on correctness, contracts, and `make test` / e2e evidence.

## Tooling and docs vs implementation

| Item | Today | Planned later |
|------|--------|----------------|
| Load / SLO proof | Estimates in [scaling.md](scaling.md) | `make bench` (e.g. ghz, k6, pgbench); record p99 and drift results in docs |
| `deploy/` (Helm/K8s) | Described in [deployment.md](deployment.md) only | Add charts/manifests under `deploy/` |
| CI image pipeline | GitHub Actions runs lint, `make test`, e2e | Build/push images; optional scan on default branch |
| `gosec` | CI security workflow | Enable in [.golangci.yml](../.golangci.yml) when tightening local lint |
| `govulncheck` / Spectral / `buf breaking` | CI + optional `make lint` | Fail `make lint` when tools missing (optional) |

## Testing and developer experience

| Item | Today | Planned later |
|------|--------|----------------|
| Run all tests | Use **`make test`** and **`make test-e2e`** | Document in README; `go test ./...` alone skips `integration` / `e2e` build tags |
| `internal/region/postgres` | Covered by `integration` tests (testcontainers) | Package-level unit tests for SQL helpers where worthwhile |
| Local lint | Requires optional binaries (`buf`, `golangci-lint`, …) | Dev container or `make deps-dev` with pinned tool installs |
| E2E | Docker Compose + multiregion override | Keep as gate; optional slimmer smoke job if runtime becomes an issue |

## Not changing the submission scope

These stay as documented assumptions unless product asks otherwise: stub `X-Customer-ID` / `x-caller`, single demo product, per-region quota only, compose dev password.