package cli

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/blackwell-systems/gcp-emulator-control-plane/internal/config"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration management",
	Long:  `Get, set, or reset configuration values.`,
}

var configGetCmd = &cobra.Command{
	Use:   "get [key]",
	Short: "Get configuration values",
	Long: `Display configuration values.

Without arguments, shows all configuration.
Specify a key to show only that value.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		display, err := config.Display()
		if err != nil {
			return err
		}

		fmt.Print(display)
		return nil
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Long: `Set a configuration value and save to config file.

Available keys:
  iam-mode         IAM mode (off|permissive|strict)
  trace            Enable trace logging (true|false)
  pull-on-start    Pull images before starting (true|false)
  policy-file      Path to policy.yaml`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		value := args[1]

		// Load current config
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		// Update based on key
		switch key {
		case "iam-mode":
			if value != "off" && value != "permissive" && value != "strict" {
				return fmt.Errorf("invalid iam-mode: %s (must be off, permissive, or strict)", value)
			}
			cfg.IAMMode = value
		case "trace":
			cfg.Trace = value == "true"
		case "pull-on-start":
			cfg.PullOnStart = value == "true"
		case "policy-file":
			cfg.PolicyFile = value
		default:
			return fmt.Errorf("unknown config key: %s", key)
		}

		// Save
		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		color.Green("✓ Configuration updated")
		fmt.Printf("\n%s: %s\n", key, value)
		color.Cyan("\nRestart the stack for changes to take effect:")
		color.Cyan("  gcp-emulator restart")

		return nil
	},
}

var configResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset configuration to defaults",
	Long:  `Reset all configuration values to their defaults.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := &config.Config{
			IAMMode:     "permissive",
			Trace:       false,
			PullOnStart: false,
			PolicyFile:  "./policy.yaml",
			Ports: config.PortConfig{
				IAM:           8080,
				SecretManager: 9090,
				KMS:           9091,
			},
		}

		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		color.Green("✓ Configuration reset to defaults")
		return nil
	},
}

func init() {
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configResetCmd)
}
