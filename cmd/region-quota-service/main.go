package main

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"quota-challenge/internal/buildinfo"
)

func main() {
	root := &cobra.Command{
		Use:   "region-quota-service",
		Short: "Per-region quota ledger and gRPC API",
	}

	serveCmd := &cobra.Command{
		Use:                "serve",
		Short:              "Run gRPC server",
		DisableFlagParsing: true,
		Run: func(cmd *cobra.Command, args []string) {
			os.Exit(runServe(args))
		},
	}

	migrateCmd := &cobra.Command{
		Use:                "migrate",
		Short:              "Apply database migrations",
		DisableFlagParsing: true,
		Run: func(cmd *cobra.Command, args []string) {
			os.Exit(runMigrate(args))
		},
	}

	reapCmd := &cobra.Command{
		Use:                "reap",
		Short:              "Expire reserved services past TTL",
		DisableFlagParsing: true,
		Run: func(cmd *cobra.Command, args []string) {
			os.Exit(runReap(args))
		},
	}

	driftCmd := &cobra.Command{
		Use:                "drift-check",
		Short:              "Recompute counter drift",
		DisableFlagParsing: true,
		Run: func(cmd *cobra.Command, args []string) {
			os.Exit(runDrift(args))
		},
	}

	healthcheckCmd := &cobra.Command{
		Use:   "healthcheck",
		Short: "Database connectivity check",
		Run: func(cmd *cobra.Command, args []string) {
			os.Exit(runHealthcheck())
		},
	}

	root.AddCommand(serveCmd, migrateCmd, reapCmd, driftCmd, healthcheckCmd)
	buildinfo.BindRoot(root)

	if err := root.Execute(); err != nil {
		slog.Error("cli", "err", err)
		os.Exit(1)
	}
}