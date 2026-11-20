# Repository layout

Follows [golang-standards/project-layout](https://github.com/golang-standards/project-layout): public contracts under `api/`, binaries under `cmd/`, private app code under `internal/`.

```
quota-challenge/
├── api/
│   ├── openapi/            # OpenAPI spec + embed for /docs
│   ├── proto/              # Protobuf sources (buf)
│   └── gen/                # Generated protobuf Go (buf generate)
├── build/                  # Dockerfiles per binary
├── cmd/                    # quota-gateway, region-quota-service CLIs
├── configs/                # otel/prometheus for Compose
├── versions.env            # pinned toolchain/image versions (Makefile, CI)
├── deployments/            # docker-compose (+ multiregion override)
├── docs/                   # Product & ops documentation
├── internal/               # Application code (not importable externally)
│   ├── capacity/
│   ├── gateway/
│   ├── regionclient/
│   ├── telemetry/
│   ├── mock/
│   └── region/             # postgres, regionsvc, grpcserver
├── scripts/                # gitlint, hooks
└── test/
    ├── integration/        # //go:build integration + testcontainers
    └── e2e/                # //go:build e2e + Compose multiregion
```

Go keeps **one package per directory**, so handler files stay under `internal/gateway/` as a single `gateway` package rather than nested subfolders.

## Tests

- **`make test`** — race-enabled unit tests plus `test/integration` (Docker for Postgres testcontainers).
- **`make test-e2e`** — multiregion Compose under `deployments/` + `test/e2e` tag.

Gaps and follow-ups: [future-work.md](future-work.md).

## Try it

- `make up` → gateway on `:8080`, docs at `/docs`
- Demo customer with seeded quotas: header `X-Customer-ID: demo`
- Product catalog: `GET /v1/regions/{region}/products` or `GET /v1/products`