package docker

import (
	"fmt"
	"os/exec"

	"github.com/blackwell-systems/gcp-emulator-control-plane/internal/config"
)

// Restart restarts the stack or a specific service
func Restart(cfg *config.Config, service *string) error {
	args := []string{"restart"}

	if service != nil {
		args = append(args, *service)
	}

	cmd := exec.Command("docker-compose", args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker-compose restart failed: %w\n%s", err, output)
	}

	return nil
}
