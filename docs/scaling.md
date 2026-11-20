# Scaling

Planning estimates only — **not measured in the submission**. A `make bench` target (ghz, k6, pgbench) and recorded results are **planned for later** ([future-work.md](future-work.md)). Design: [design.md](design.md).

**Ceiling (unsharded, step 0):** ~150–250 creates/s per region when all creates hit the same pool rows. Bursts of 50–100/s have ~2–4× headroom. Correctness does not depend on load; hot rows affect latency and throughput only.

**Scale-out:** add regions = new DB + region service + gateway registry entry. Replicas are stateless; one Postgres primary per region is the writer.

## Planning snapshot

| Input | Value |
|---|---|
| Create shape | VM: 4 lines (cpu, ram, disk, public_ip) |
| Typical load | 1–5 creates/s avg; 50–100/s bursts; reads ~10× writes |

One create ≈ 16 SQL ops, ~8–12 ms DB time with sync replica (dominates p99 under SLO).

## Main bottlenecks

| Limiting factor | Mitigation in design |
|---|---|
| Hot `pools` / `customer_quotas` rows | Quota updates → inserts → pool updates (short pool lock); optional pool sharding (step 2) |
| MVCC churn on counter rows | `fillfactor`, autovacuum, short transactions |
| Sold-out retry storms | Gateway rate limit, reclaim singleflight, `Retry-After` |
| Connections | PgBouncer (prod) |
| Reads | CNPG read replica + short fan-out cache |

Full stage-by-stage table lives in git history if needed; operational focus is **pool row ordering** and **rejection storm** controls above.

## Scaling ladder

| Step | Change | ~Creates/s per region |
|---|---|---|
| 0 | As designed | 150–250 |
| 1 | PgBouncer, read replicas for GET, gateway cache/rate limits | Same writes, stable p99 |
| 2 | Shard abundant pools (`FOR UPDATE SKIP LOCKED` per shard) | 1k–1.5k |
| 3 | Bigger primary / faster disk | ~2× |
| 4 | Cells (split region into multiple DBs) | Linear; not planned |

Scarce resources (e.g. public_ip) stay on a single counter; shard only abundant types (ram, disk).

## Verification

Benchmark scenarios (for when `make bench` exists): spread load, hot pool, sold-out storm, mass expiry, primary failover. **Target acceptance:** p99 &lt; 300 ms at 100 creates/s for 10 min; zero drift after dry-run resync — verify later and replace the planning numbers in this doc.