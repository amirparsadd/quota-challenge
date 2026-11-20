package main

import (
	"os"

	"github.com/spf13/cobra"

	"quota-challenge/internal/buildinfo"
)

func main() {
	root := &cobra.Command{
		Use:   "quota-gateway",
		Short: "Public REST gateway for regional quota services",
		Run: func(cmd *cobra.Command, args []string) {
			os.Exit(run())
		},
	}

	serveCmd := &cobra.Command{
		Use:   "serve",
		Short: "Run HTTP server (default when invoked with no subcommand)",
		Run: func(cmd *cobra.Command, args []string) {
			os.Exit(run())
		},
	}

	var live bool
	healthcheckCmd := &cobra.Command{
		Use:   "healthcheck",
		Short: "Probe ops /readyz (or /livez with --live)",
		Run: func(cmd *cobra.Command, args []string) {
			runHealthcheckCmd(live)
		},
	}
	healthcheckCmd.Flags().BoolVar(&live, "live", false, "Check /livez instead of /readyz")

	root.AddCommand(serveCmd, healthcheckCmd)
	buildinfo.BindRoot(root)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}