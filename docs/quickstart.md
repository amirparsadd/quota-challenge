# Quickstart

Use the **Quota API Gateway** on port **8080**. The **multiregion** compose stack seeds customer **`demo`**, product **`vm`**, and regions **`ir-thr-1`**, **`ir-thr-2`**, and **`ir-tbz-1`** (fan-out and per-region APIs work out of the box).

## 1. Run locally

**Prerequisites:** Docker Desktop (or Docker Engine), `make`.

```bash
git clone <repo>
cd quota-challenge
cp .env.example .env
make up-multiregion   # gateway + 3 regions + Postgres (compose + multiregion override)
# Lighter single-region stack: make up / make down
```

Check health (gateway **ops port** 8081 is internal in Compose; Prometheus UI on **9091**):

```bash
docker compose exec quota-api-gateway /quota-gateway healthcheck
curl -s -o /dev/null -w "%{http_code}" http://127.0.0.1:9091/-/ready
```

Stop and wipe data:

```bash
make down-multiregion    # stops multiregion stack and removes volumes
```

**API reference (Scalar):** [http://localhost:8080/docs](http://localhost:8080/docs)  
**OpenAPI file:** [http://localhost:8080/openapi/v1.yaml](http://localhost:8080/openapi/v1.yaml)

## 2. Headers you need

| Header | Required | Notes |
|---|---|---|
| `X-Customer-ID` | On almost all `/v1/*` calls | Use **`demo`** in compose (quotas are seeded) |
| `Idempotency-Key` | **POST** `.../services` only | Client-generated UUID; **reuse on retry** |
| `Content-Type: application/json` | POST/PATCH with body | |

Auth is a stub in this project—there is no API key beyond `X-Customer-ID`. See [assumptions.md](assumptions.md).

## 3. Discover products and quota

What you can request:

```bash
curl -s http://localhost:8080/v1/regions/ir-thr-1/products \
  -H 'X-Customer-ID: demo' | jq
```

Your limits and usage in the region:

```bash
curl -s http://localhost:8080/v1/regions/ir-thr-1/quota \
  -H 'X-Customer-ID: demo' | jq
```

Advisory remaining capacity (create is still the real guarantee):

```bash
curl -s 'http://localhost:8080/v1/regions/ir-thr-1/availability?product=vm' \
  -H 'X-Customer-ID: demo' | jq
```

HTTP errors use **RFC 9457** `application/problem+json`. Slugs are documented at:

```bash
curl -s http://localhost:8080/v1/problems | jq
curl -s http://localhost:8080/v1/problems/validation-error | jq
```

## 4. Create a reservation

Standard **vm** shape: `cpu`, `ram`, `disk`, `public_ip`.

```bash
export IDEM="$(uuidgen)"
curl -s -D - -X POST http://localhost:8080/v1/regions/ir-thr-1/services \
  -H 'X-Customer-ID: demo' \
  -H "Idempotency-Key: $IDEM" \
  -H 'Content-Type: application/json' \
  -d '{
    "product": "vm",
    "items": [
      {"resource": "cpu", "amount": 2},
      {"resource": "ram", "amount": 4},
      {"resource": "disk", "amount": 40},
      {"resource": "public_ip", "amount": 1}
    ]
  }'
```

- **`201 Created`** — body includes `service` (`state` is usually `reserved`; default TTL applies).
- **`Location`** header — path to `GET .../services/{id}`.
- **Retry** the same request with the **same** `Idempotency-Key` — you get the same outcome (`Idempotent-Replayed: true` on replay).

Omit `ttl_seconds` for the product default (~10 minutes). Use **`"ttl_seconds": 0`** to create **active** immediately (no confirm step).

## 5. Confirm or cancel

Replace `{id}` with the service UUID from create or `Location`.

**Confirm** (`reserved` → `active`):

```bash
curl -s -X PATCH "http://localhost:8080/v1/regions/ir-thr-1/services/{id}" \
  -H 'X-Customer-ID: demo' \
  -H 'Content-Type: application/json' \
  -d '{"state":"active"}'
```

**Cancel** a reservation:

```bash
curl -s -X DELETE "http://localhost:8080/v1/regions/ir-thr-1/services/{id}" \
  -H 'X-Customer-ID: demo'
```

**List** services:

```bash
curl -s http://localhost:8080/v1/regions/ir-thr-1/services \
  -H 'X-Customer-ID: demo' | jq
```

**History:**

```bash
curl -s "http://localhost:8080/v1/regions/ir-thr-1/services/{id}/events" \
  -H 'X-Customer-ID: demo' | jq
```

## 6. Multiregion fan-out

With `make up-multiregion` already running, aggregate reads hit every healthy region in parallel:

```bash
curl -s http://localhost:8080/v1/availability -H 'X-Customer-ID: demo' | jq
curl -s http://localhost:8080/v1/quotas       -H 'X-Customer-ID: demo' | jq
curl -s http://localhost:8080/v1/services    -H 'X-Customer-ID: demo' | jq
```

If a region is down, responses include `"partial": true` and `unavailable_regions`.

## 7. Develop and verify

From the repo root (Go 1.22+):

```bash
make test        # unit + integration (Testcontainers Postgres)
make lint        # buf lint, optional golangci/spectral
make test-e2e    # full multiregion Docker e2e
```

## Where to go next

| Doc | Use when |
|---|---|
| [design.md](design.md) | Lifecycle, idempotency, counters, APIs |
| [deployment.md](deployment.md) | Compose layout, OTel, production notes |
| [structure.md](structure.md) | Repository layout |
| [requirements.md](requirements.md) | Acceptance criteria and test map |