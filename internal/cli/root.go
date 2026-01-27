package cli

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gcp-emulator",
	Short: "Manage the GCP Emulator Control Plane",
	Long: `gcp-emulator is a unified CLI for managing the GCP Emulator Control Plane.

It orchestrates IAM, Secret Manager, and KMS emulators with centralized
authorization policy.`,
	SilenceUsage: true,
}

// Execute runs the root command
func Execute(version string) error {
	rootCmd.Version = version
	return rootCmd.Execute()
}

func init() {
	// Add subcommands
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(restartCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(logsCmd)
	rootCmd.AddCommand(policyCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(versionCmd)
}
