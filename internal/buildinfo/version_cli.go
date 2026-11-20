package buildinfo

import (
	"fmt"

	"github.com/spf13/cobra"
)

// VersionCommand prints build metadata.
func VersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version and Git commit",
		Run: func(*cobra.Command, []string) {
			fmt.Println(String())
		},
	}
}

// BindRoot sets --version output on a Cobra root command.
func BindRoot(cmd *cobra.Command) {
	cmd.Version = String()
	cmd.SetVersionTemplate("{{.Version}}\n")
	cmd.AddCommand(VersionCommand())
}