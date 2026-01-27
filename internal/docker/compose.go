// Package docker provides a wrapper around docker compose commands for managing
// the GCP emulator stack. It abstracts compose operations and injects configuration
// via environment variables.
package docker

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/blackwell-systems/gcp-emulator-control-plane/internal/config"
)

// getComposeCommand returns the appropriate docker compose command
// Tries "docker compose" first (modern), falls back to "docker-compose" (legacy)
func getComposeCommand() (string, []string) {
	// Try modern "docker compose" first
	cmd := exec.Command("docker", "compose", "version")
	if err := cmd.Run(); err == nil {
		return "docker", []string{"compose"}
	}
	
	// Fall back to legacy "docker-compose"
	return "docker-compose", []string{}
}

// Start starts the docker compose stack
func Start(cfg *config.Config) error {
	// Generate environment variables for docker compose
	env := os.Environ()
	env = append(env, 
		fmt.Sprintf("IAM_MODE=%s", cfg.IAMMode),
		fmt.Sprintf("IAM_PORT=%d", cfg.Ports.IAM),
		fmt.Sprintf("SECRET_MANAGER_PORT=%d", cfg.Ports.SecretManager),
		fmt.Sprintf("KMS_PORT=%d", cfg.Ports.KMS),
	)

	// Get appropriate compose command
	binary, baseArgs := getComposeCommand()
	args := append(baseArgs, "up", "-d")
	
	// Run docker compose up
	cmd := exec.Command(binary, args...)
	cmd.Env = env

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker compose up failed: %w\n%s", err, output)
	}

	return nil
}

// Stop stops the docker compose stack
func Stop() error {
	binary, baseArgs := getComposeCommand()
	args := append(baseArgs, "down")
	
	cmd := exec.Command(binary, args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker compose down failed: %w\n%s", err, output)
	}

	return nil
}

// Pull pulls the latest images
func Pull() error {
	binary, baseArgs := getComposeCommand()
	args := append(baseArgs, "pull")
	
	cmd := exec.Command(binary, args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker compose pull failed: %w\n%s", err, output)
	}

	return nil
}
