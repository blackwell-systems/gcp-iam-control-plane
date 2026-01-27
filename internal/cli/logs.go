package cli

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var (
	logsFollow bool
	logsTail   int
	logsSince  string
)

var logsCmd = &cobra.Command{
	Use:   "logs [service]",
	Short: "Show logs from services",
	Long: `Show logs from emulator services.

Without arguments, shows logs from all services.
Specify a service name to show logs from that service only.

Services: iam, secret-manager, kms`,
	ValidArgs: []string{"iam", "secret-manager", "kms"},
	RunE: func(cmd *cobra.Command, args []string) error {
		args = buildLogsArgs(args)

		dcCmd := exec.Command("docker-compose", args...)
		dcCmd.Stdout = os.Stdout
		dcCmd.Stderr = os.Stderr

		return dcCmd.Run()
	},
}

func buildLogsArgs(services []string) []string {
	args := []string{"logs"}

	if logsFollow {
		args = append(args, "--follow")
	}

	if logsTail > 0 {
		args = append(args, "--tail", fmt.Sprintf("%d", logsTail))
	}

	if logsSince != "" {
		args = append(args, "--since", logsSince)
	}

	// Add service names if specified
	if len(services) > 0 {
		args = append(args, services...)
	}

	return args
}

func init() {
	logsCmd.Flags().BoolVarP(&logsFollow, "follow", "f", false, "Follow log output")
	logsCmd.Flags().IntVar(&logsTail, "tail", 50, "Number of lines to show from end of logs")
	logsCmd.Flags().StringVar(&logsSince, "since", "", "Show logs since timestamp (e.g. 2m, 1h)")
}
