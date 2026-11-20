# Technologies

Stack inventory for threat modeling and dependency review. Scope: [assumptions.md](assumptions.md).

## Components

| Piece | Tech | Exposure |
|---|---|---|
| Gateway | Go, `quota-gateway` | Internet :8080 |
| Region service | Go, `region-quota-service` | Internal gRPC :9090 |
| Database | PostgreSQL 16.16-bookworm (CNPG in prod ref) | Internal |
| Submission runtime | Docker Compose | Local / CI |
| Contracts | OpenAPI 3, protobuf + buf | Repo artifacts |

**Submission auth (weak by design):** `X-Customer-ID`, `x-caller` — see assumptions.

## Trust boundaries

| From → To | Control |
|---|---|
| Internet → gateway | TLS (ingress in prod); stub header today |
| Gateway → region | gRPC (mTLS in prod ref) |
| Region → Postgres | Network + DB auth; parameterized SQL |
| CI → images | GitHub Actions; scan on `main` |

## Data sensitivity

| Store | Contents |
|---|---|
| PostgreSQL | customer_id, quotas, services, idempotency bodies, events |
| Gateway | None durable |
| Logs/traces | customer_id, service_id possible — no request bodies |

## Dependencies (Go)

`pgx`, gRPC/protobuf, `spf13/cobra`, OpenTelemetry, `goose`; tests: `testify`, `go-cmp`.

## Threat modeling entry points

1. Public REST `/v1/*` — abuse, IDOR, idempotency replay
2. Internal gRPC + **CapacityAdmin** — privilege escalation
3. SQL via app — injection if queries wrong
4. Compose dev credentials — local-only risk

**Priority assets:** pool/quota counters, idempotency store, DB backups.

## Out of scope (no integration)

Billing, Dejban, IaaS, message bus, Redis, Vault in compose.

Update this file when adding dependencies, entry points, or auth changes.