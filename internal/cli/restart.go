package cli

import (
	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/blackwell-systems/gcp-emulator-control-plane/internal/config"
	"github.com/blackwell-systems/gcp-emulator-control-plane/internal/docker"
)

var restartCmd = &cobra.Command{
	Use:   "restart [service]",
	Short: "Restart the emulator stack",
	Long: `Restart all services or a specific service.

Without arguments, restarts entire stack.
Specify a service name to restart only that service.

Services: iam, secret-manager, kms`,
	ValidArgs: []string{"iam", "secret-manager", "kms"},
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		if len(args) == 0 {
			color.Cyan("Restarting entire stack...")
			if err := docker.Restart(cfg, nil); err != nil {
				color.Red("✗ Failed to restart: %v", err)
				return err
			}
			color.Green("✓ Stack restarted successfully")
		} else {
			service := args[0]
			color.Cyan("Restarting %s...", service)
			if err := docker.Restart(cfg, &service); err != nil {
				color.Red("✗ Failed to restart %s: %v", service, err)
				return err
			}
			color.Green("✓ %s restarted successfully", service)
		}

		return nil
	},
}
