.PHONY: up down up-multiregion down-multiregion deps test test-e2e lint proto migrate build install-hooks lint-commits

COMPOSE ?= docker compose --project-directory $(CURDIR)
COMPOSE_FILE ?= deployments/docker-compose.yml
E2E_COMPOSE ?= $(COMPOSE) -f deployments/docker-compose.yml -f deployments/docker-compose.multiregion.yml

include versions.env
export

# Load .env for local targets when present (Compose also reads .env for substitution).
ifneq (,$(wildcard .env))
include .env
export
endif

up:
	$(COMPOSE) -f $(COMPOSE_FILE) up -d --build --wait

down:
	$(COMPOSE) -f $(COMPOSE_FILE) down -v

up-multiregion:
	$(E2E_COMPOSE) up -d --build --wait

down-multiregion:
	$(E2E_COMPOSE) down -v

# Distroless runtime base hardcoded in build/*/Dockerfile (Shecan gcr mirror).
build:
	docker build -f build/quota-gateway/Dockerfile \
		--build-arg GOLANG_IMAGE=$(GOLANG_IMAGE) \
		-t quota-gateway:local .
	docker build -f build/region-quota-service/Dockerfile \
		--build-arg GOLANG_IMAGE=$(GOLANG_IMAGE) \
		-t region-quota-service:local .

deps:
	go mod tidy
	go mod download

proto:
	buf lint
	buf generate

proto-docker:
	docker run --rm -v "$(CURDIR):/workspace" -w /workspace \
		-v "$(shell go env GOPATH)/bin:/go/bin" \
		-e PATH="/go/bin:$$PATH" \
		$(BUF_IMAGE) generate

migrate:
	go run ./cmd/region-quota-service migrate --database-url "$${DATABASE_URL:-postgres://quota:dev@localhost:5432/quota?sslmode=disable}"

test:
	go test -race -count=1 ./internal/capacity/... ./internal/region/regionsvc/... ./internal/region/postgres/... ./internal/gateway/...
	go test -race -count=1 -p 1 -tags=integration ./test/integration/...

test-e2e:
	$(E2E_COMPOSE) up -d --build --wait
	go test -race -count=1 -tags=e2e -timeout 5m ./test/e2e/... || ( $(E2E_COMPOSE) logs; exit 1 )
	$(E2E_COMPOSE) down -v

lint:
	@command -v golangci-lint >/dev/null 2>&1 && golangci-lint run ./... || echo "skip: golangci-lint not installed"
	@command -v buf >/dev/null 2>&1 && buf lint || echo "skip: buf not installed (see versions.env BUF_VERSION; CI installs buf)"
	@command -v buf >/dev/null 2>&1 && { buf breaking --against '.git#branch=main' 2>/dev/null || buf breaking --against '.git#branch=master' 2>/dev/null || echo "skip: buf breaking (no baseline branch)"; } || true
	@test -f api/openapi/v1.yaml
	@command -v npx >/dev/null 2>&1 && npx --yes @stoplight/spectral-cli@$(SPECTRAL_CLI_VERSION) lint api/openapi/v1.yaml 2>/dev/null || echo "skip: spectral not available"

install-hooks:
	./scripts/install-git-hooks.sh

lint-commits:
	./scripts/lint-commits.sh