# Cloud Capacity Allocation System

Can this customer get this capacity **right now**, and what is the state of their request?

- **Region Quota Service** — gRPC, one per region; owns capacity, limits, and services in that region's PostgreSQL.
- **Quota API Gateway** — public REST; routes to regions, fan-out across regions.

Reservations have a **TTL**; confirm clears it and moves capacity to **allocated**. Creates use **idempotency keys** and atomic counter updates on the ledger.

## Quick start

```bash
cp .env.example .env
make up    # gateway on http://localhost:8080 (bind/port from .env)
```

Step-by-step usage (products, quota, create, confirm, cancel): **[docs/quickstart.md](docs/quickstart.md)**.  
API browser: [http://localhost:8080/docs](http://localhost:8080/docs).

Compose details: [deployment.md](docs/deployment.md).

## Docs

| Doc | Contents |
|---|---|
| [quickstart.md](docs/quickstart.md) | Run compose, curl examples, typical flows |
| [assumptions.md](docs/assumptions.md) | Scope and fixed decisions |
| [requirements.md](docs/requirements.md) | Goals, SLOs, acceptance tests |
| [design.md](docs/design.md) | Architecture, data model, APIs, algorithms |
| [structure.md](docs/structure.md) | Source tree (`internal/`, `cmd/`, tests) |
| [deployment.md](docs/deployment.md) | Compose + production reference |
| [scaling.md](docs/scaling.md) | Throughput estimates and ladder |
| [technologies.md](docs/technologies.md) | Stack for threat modeling |
| [future-work.md](docs/future-work.md) | Benchmarks, `deploy/`, CI/images, tests — planned after submission |

Contracts: [api/openapi/v1.yaml](api/openapi/v1.yaml), [api/proto/regionquota/v1/](api/proto/regionquota/v1/).