package main

import (
	"context"
	"log/slog"
	"net"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	"quota-challenge/internal/obslog"
	"quota-challenge/internal/region/grpcserver"
	"quota-challenge/internal/region/postgres"
	"quota-challenge/internal/region/regionsvc"
	"quota-challenge/internal/telemetry"
)

func runServe(args []string) int {
	obslog.SetupDefault("region-quota-service")
	regionID := env("REGION_ID", "")
	if regionID == "" {
		slog.Error("REGION_ID required")
		return 1
	}
	dsn := env("DATABASE_URL", "")
	if dsn == "" {
		slog.Error("DATABASE_URL required")
		return 1
	}
	grpcAddr := flagString(args, "--grpc-addr", env("GRPC_ADDR", ":9090"))
	autoMigrate := flagBool(args, "--auto-migrate")
	seedDemo := flagBool(args, "--seed-demo")
	devReaper := flagDuration(args, "--dev-reaper")

	ctx := context.Background()
	shutdown, err := telemetry.Init(ctx, "region-quota-service")
	if err != nil {
		slog.Error("telemetry", "err", err)
		return 1
	}
	defer func() { _ = shutdown(context.Background()) }()

	if autoMigrate {
		if err := postgres.Migrate(ctx, dsn); err != nil {
			slog.Error("migrate", "err", err)
			return 1
		}
	}
	poolSettings := poolSettingsFromEnv()
	pool, err := postgres.OpenPool(ctx, dsn, poolSettings)
	if err != nil {
		slog.Error("db", "err", err)
		return 1
	}
	var poolOnce sync.Once
	closePool := func() {
		poolOnce.Do(func() {
			pool.Close()
			slog.Info("db pool closed")
		})
	}
	defer closePool()
	if poolSettings.SetMaxConns || poolSettings.SetMinConns || poolSettings.SetMaxConnLifetime || poolSettings.SetMaxConnIdleTime {
		slog.Info("db pool configured",
			"max_conns", pool.Config().MaxConns,
			"min_conns", pool.Config().MinConns,
			"max_conn_lifetime", pool.Config().MaxConnLifetime,
			"max_conn_idle_time", pool.Config().MaxConnIdleTime,
		)
	}
	if seedDemo {
		if err := postgres.SeedDemo(ctx, pool, regionID); err != nil {
			slog.Error("seed", "err", err)
			return 1
		}
	}
	store := postgres.NewStore(pool, regionID)
	if err := store.AssertRegion(ctx); err != nil {
		if seedDemo {
			_ = store.EnsureRegionMeta(ctx)
			_ = postgres.SeedDemo(ctx, pool, regionID)
		}
		if err2 := store.AssertRegion(ctx); err2 != nil {
			slog.Error("region assert", "err", err2)
			return 1
		}
	}
	svc := regionsvc.NewWithConfig(store, regionConfigFromEnv())
	if telemetry.Global != nil {
		telemetry.Global.RegisterPoolUtilization(func(c context.Context) []telemetry.PoolUtilSample {
			return store.PoolUtilizationSamples(c)
		})
		go poolMetricsLoop(store)
	}

	if devReaper > 0 {
		go startDevReaper(svc, devReaper)
	}

	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		slog.Error("listen", "err", err)
		return 1
	}
	gs := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
			grpc.ChainUnaryInterceptor(
			obslog.UnaryServerTimeoutInterceptor(dbQueryTimeoutFromEnv()),
			obslog.UnaryServerInterceptor(),
		),
	)
	grpcserver.Register(gs, svc, regionID)
	healthpb.RegisterHealthServer(gs, health.NewServer())

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		slog.Info("shutdown", "signal", ctx.Err())
		gs.GracefulStop()
		closePool()
	}()
	slog.Info("serving gRPC", "addr", grpcAddr, "region", regionID)
	if err := gs.Serve(lis); err != nil {
		slog.Error("serve", "err", err)
		return 1
	}
	return 0
}

func poolMetricsLoop(store *postgres.Store) {
	t := time.NewTicker(30 * time.Second)
	defer t.Stop()
	for range t.C {
		if telemetry.Global != nil {
			telemetry.Global.RecordPoolUtilization(context.Background())
		}
	}
}

func startDevReaper(svc *regionsvc.Service, every time.Duration) {
	t := time.NewTicker(every)
	defer t.Stop()
	for range t.C {
		if n, err := svc.Reap(context.Background()); err != nil {
			slog.Warn("reap", "err", err)
		} else if n > 0 {
			slog.Info("reaped", "count", n)
		}
	}
}

func mustPool(dsn string) *pgxpool.Pool {
	pool, err := postgres.OpenPool(context.Background(), dsn, postgres.PoolSettings{})
	if err != nil {
		slog.Error("db", "err", err)
		panic("db")
	}
	return pool
}