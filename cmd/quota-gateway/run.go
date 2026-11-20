package main

import (
	"context"
	"log/slog"
	"net/http"
	"os/signal"
	"syscall"

	"quota-challenge/internal/gateway"
	"quota-challenge/internal/obslog"
	"quota-challenge/internal/regionclient"
	"quota-challenge/internal/telemetry"
)

func run() int {
	obslog.SetupDefault("quota-gateway")
	ctx := context.Background()
	shutdown, err := telemetry.Init(ctx, "quota-gateway")
	if err != nil {
		slog.Error("telemetry", "err", err)
		return 1
	}
	defer func() { _ = shutdown(context.Background()) }()

	addr := listenHTTPAddr()
	regions := parseRegions(env("REGIONS", "ir-thr-1=localhost:9090"))

	backends := make(map[string]*regionclient.Client)
	for id, target := range regions {
		c, err := regionclient.Dial(context.Background(), id, target)
		if err != nil {
			slog.Error("dial region", "region", id, "err", err)
			return 1
		}
		defer c.Close()
		backends[id] = c
	}

	gwTimeouts := gatewayTimeoutsFromEnv()
	readHeader, shutdownTimeout := httpServerTimeoutsFromEnv()
	srv := gateway.NewServer(backends, gwTimeouts)
	httpSrv := &http.Server{Addr: addr, Handler: srv.Handler(), ReadHeaderTimeout: readHeader}

	opsAddr := listenOpsAddr()
	opsSrv := &http.Server{
		Addr:              opsAddr,
		Handler:           srv.OpsHandler(telemetry.PrometheusHandler()),
		ReadHeaderTimeout: readHeader,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		_ = opsSrv.Shutdown(shutdown)
		_ = httpSrv.Shutdown(shutdown)
	}()

	go func() {
		slog.Info("gateway ops listening", "addr", opsAddr)
		if err := opsSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("ops serve", "err", err)
			stop()
		}
	}()

	slog.Info("gateway listening", "addr", addr, "ops", opsAddr, "regions", len(backends))
	if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("serve", "err", err)
		return 1
	}
	return 0
}