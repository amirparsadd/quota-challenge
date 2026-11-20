-- +goose Up
CREATE TABLE region_meta (
    region_id TEXT PRIMARY KEY,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE resource_types (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    unit TEXT NOT NULL,
    min_amount BIGINT NOT NULL DEFAULT 1,
    step_amount BIGINT NOT NULL DEFAULT 1,
    max_amount BIGINT,
    discrete BOOLEAN NOT NULL DEFAULT false
);

CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    default_ttl_seconds INT NOT NULL DEFAULT 600,
    max_ttl_seconds INT NOT NULL DEFAULT 3600
);

CREATE TABLE product_resources (
    product_id INT NOT NULL REFERENCES products(id),
    resource_type_id INT NOT NULL REFERENCES resource_types(id),
    PRIMARY KEY (product_id, resource_type_id)
);

CREATE TABLE pools (
    resource_type_id INT PRIMARY KEY REFERENCES resource_types(id),
    total BIGINT NOT NULL,
    baseline_allocated BIGINT NOT NULL DEFAULT 0,
    allocated BIGINT NOT NULL DEFAULT 0,
    reserved BIGINT NOT NULL DEFAULT 0,
    CHECK (allocated + reserved <= total)
);

CREATE TABLE customer_quotas (
    customer_id TEXT NOT NULL,
    resource_type_id INT NOT NULL REFERENCES resource_types(id),
    quota_limit BIGINT NOT NULL DEFAULT 0,
    allocated BIGINT NOT NULL DEFAULT 0,
    reserved BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (customer_id, resource_type_id),
    CHECK (quota_limit = 0 OR allocated + reserved <= quota_limit)
);

CREATE TABLE services (
    id UUID PRIMARY KEY,
    customer_id TEXT NOT NULL,
    product_id INT NOT NULL REFERENCES products(id),
    state TEXT NOT NULL,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX services_customer_idx ON services (customer_id);
CREATE INDEX services_state_expires_idx ON services (state, expires_at) WHERE state = 'reserved';

CREATE TABLE service_resources (
    service_id UUID NOT NULL REFERENCES services(id) ON DELETE CASCADE,
    resource_type_id INT NOT NULL REFERENCES resource_types(id),
    amount BIGINT NOT NULL,
    PRIMARY KEY (service_id, resource_type_id)
);

CREATE TABLE service_events (
    id BIGSERIAL PRIMARY KEY,
    service_id UUID NOT NULL REFERENCES services(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE idempotency_keys (
    key TEXT NOT NULL,
    customer_id TEXT NOT NULL,
    request_hash TEXT NOT NULL,
    status TEXT NOT NULL,
    lease_expires_at TIMESTAMPTZ,
    lease_token UUID,
    response_status INT,
    response_body JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (customer_id, key)
);

CREATE TABLE counter_resyncs (
    id BIGSERIAL PRIMARY KEY,
    dry_run BOOLEAN NOT NULL,
    drift BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS counter_resyncs;
DROP TABLE IF EXISTS idempotency_keys;
DROP TABLE IF EXISTS service_events;
DROP TABLE IF EXISTS service_resources;
DROP TABLE IF EXISTS services;
DROP TABLE IF EXISTS customer_quotas;
DROP TABLE IF EXISTS pools;
DROP TABLE IF EXISTS product_resources;
DROP TABLE IF EXISTS products;
DROP TABLE IF EXISTS resource_types;
DROP TABLE IF EXISTS region_meta;