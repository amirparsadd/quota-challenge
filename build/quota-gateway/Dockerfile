# syntax=docker/dockerfile:1.13

# GOLANG_IMAGE default matches versions.env; override via build-arg.
ARG GOLANG_IMAGE=1.26.11-bookworm

FROM golang:${GOLANG_IMAGE} AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/quota-gateway ./cmd/quota-gateway

# Distroless runtime: Shecan gcr.io mirror (same path as gcr.io/distroless/...).
FROM mirror-gcr.pkg.shecan.ir/distroless/static-debian12:nonroot
COPY --from=build --chown=65532:65532 /out/quota-gateway /quota-gateway
USER 65532:65532
ENTRYPOINT ["/quota-gateway"]