# Assumptions

Scope for this project. Details in [design.md](design.md).

## Submission (fixed)

| Topic | Decision |
|---|---|
| Deliverable | **Docker Compose** — working gateway + region service + Postgres; docs and contracts |
| Regions | Default compose: **one region** (`ir-thr-1`). **`docker-compose.multiregion.yml`** for e2e / fan-out tests |
| Demo product | **`vm` only** — cpu, ram, disk, public_ip |
| Customer auth | **`X-Customer-ID` header stub** (no JWT/API key in submission) |
| Internal auth | **`x-caller` gRPC metadata stub** (mTLS documented for production, not implemented here) |
| Reservation TTL | **10 minutes** default; per-product max in DB |
| CI | **GitHub Actions** in repo |
| Secrets in compose | Local values in **`.env`** (from `.env.example`); not committed |
| Secrets in production | **Documented:** Vault + ESO — not required in submission |

## Domain

1. One **service** = one **region**; multi-region ⇒ multiple services.
2. Create is **all-or-nothing** across resource lines (VM = cpu + ram + disk + ip).
3. **Quota limits are per region only** — no cross-region enforcement on the create path.
4. No limit configured in a region ⇒ limit **0** ⇒ `quota-exceeded`.
5. Resources: **divisible** (cpu, ram, disk) or **discrete** (public_ip, gpu) — integers; validation via min/step/max on `resource_types`.
6. **No per-product sale caps** — only pool capacity + per-customer quota per resource.
7. **TTL:** `expires_at` set ⇒ reservation (`reserved`); cleared ⇒ permanent (`active`). Wall-clock expiry; downtime does not extend TTL.
8. **Real VM provisioning is out of scope** — internal `Release` stands in for “service deleted”.
9. **Units** are fixed per `resource_type`; amounts are integers in storage and in REST JSON (**numbers**, not strings — see [design.md](design.md#rest-json-vs-grpc)); unit in API responses.
10. **Standalone** — no billing, Dejban, IaaS, or pub/sub in this project ([out of scope](#out-of-scope)).

## Out of scope

Billing, Dejban/auth integration, IaaS/Abrak inventory sync, event bus to other systems, Redis, and implementing Vault/mTLS/Kubernetes in the submission artifact. Production patterns for those appear only as short notes in [deployment.md](deployment.md) and [technologies.md](technologies.md).