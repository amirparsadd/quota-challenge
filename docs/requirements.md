# Requirements

Goals and acceptance criteria. Design: [design.md](design.md). Scope: [assumptions.md](assumptions.md).

## Goals

1. **Atomic reservation** — create is the guarantee; availability GET is advisory.
2. **Idempotent creates** — same key ⇒ same outcome, including stored rejections.
3. **Bounded in-flight work** — TTL + reaper (+ on-demand reclaim under contention).
4. **Visible state** — lifecycle + `service_events` audit log.
5. **Regional isolation** — one sick region ⇒ `503` / `partial: true` on fan-out, not global failure.

## SLOs (per region)

| SLI | Target |
|---|---|
| Create success rate (`POST .../services`, 409 not counted as failure) | 99.9% / 30d |
| Create latency p99 (inside region) | < 300 ms |
| Expiry lag p99 (reaper every minute) | < 90 s |
| Counter drift after dry-run resync | 0 |

## Acceptance tests

Must pass in `make test` / CI (integration + e2e where noted).

| Area | What we prove |
|---|---|
| **Capacity & quota** | Over-pool concurrency (exactly N succeed); customer quota cap; per-region isolation; lowered limit below usage; multi-line atomic rollback |
| **Lifecycle** | Expiry, confirm (`reserved`→`active`), direct `ttl=0` active, release path |
| **Idempotency** | Concurrent duplicates; payload hash mismatch; replay + rejection durability; crash after claim + lease takeover / fencing |
| **Concurrency** | Lock order: no deadlocks across create/confirm/cancel/expire/reclaim/resync |
| **Resync** | Dry-run vs apply; concurrent creates safe |
| **Recovery** | Region service restart; gateway stateless (e2e) |
| **Gateway** | Timeout after commit + retry; multiregion: B down, A works, fan-out partial (e2e) |
| **Contracts** | OpenAPI + problem+json; `buf lint` / breaking; gateway denied admin RPCs |
| **HTTP semantics** | PATCH confirm / DELETE cancel idempotent; invalid transitions ⇒ `409`/`422` |
| **Reaper & reclaim** | Double reap safe; on-demand reclaim without waiting for cron |
| **Validation** | min/step/max ⇒ `422`; `unit` on amounts |

**Load SLOs (later):** p99 &lt; 300 ms at 100 creates/s for 10 min; zero drift after dry-run resync — targets in [scaling.md](scaling.md); proof via benchmarks is **not part of the current submission** ([future-work.md](future-work.md)).

## Verification map (submission)

| Requirement row | Evidence |
|---|---|
| Capacity & quota | `test/integration/capacity_acceptance_test.go`, `test/integration/region_isolation_test.go` |
| Lifecycle | `test/integration/lifecycle_test.go`, `test/integration/store_test.go` |
| Idempotency | `test/integration/*idempotency*`, concurrent + lease tests |
| Resync | `test/integration/quota_resync_test.go` |
| Gateway / multiregion | `test/e2e/acceptance_test.go`, `test/e2e/gateway_test.go` |
| Gateway timeout → retry | `internal/gateway/grpcmap_test.go` (504), `test/e2e/idempotency_retry_test.go` |
| Recovery | `test/e2e/recovery_test.go` |
| Contracts | `api/openapi/v1.yaml`, `buf lint`, `internal/region/grpcserver/auth_test.go`, `test/e2e/docs_test.go` |
| HTTP semantics | `test/e2e/acceptance_test.go`, lifecycle idempotent confirm/cancel |

**Not automated in CI (documented only):** injecting real gRPC deadline during in-flight commit (use 504 mapping + idempotency replay tests above); formal deadlock stress across all paths; load SLOs in [scaling.md](scaling.md). Additional testing and CI hardening are tracked under [future-work.md](future-work.md).