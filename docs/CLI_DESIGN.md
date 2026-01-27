# CLI Design: gcp-emulator

Unified command-line interface for managing the GCP Emulator Control Plane.

---

## Overview

**Goal:** Provide a single CLI tool that makes the entire emulator mesh easy to use without requiring users to interact directly with docker-compose or policy files.

**Binary name:** `gcp-emulator`

**Framework:** Cobra (github.com/spf13/cobra)
- Industry standard (kubectl, docker, gh, helm use it)
- Excellent subcommand support
- Built-in help generation
- Shell completion
- Persistent flags

**Configuration:** Viper (github.com/spf13/viper)
- Automatic precedence: flags > env vars > config file > defaults
- Multiple config file formats (YAML, JSON, TOML)
- Environment variable binding
- Live config watching (optional)
- Works seamlessly with Cobra

**Output styling:** fatih/color
- Cross-platform colored output
- Auto-detects TTY
- Respects NO_COLOR environment variable
- Simple API

---

## Command Structure

```
gcp-emulator
├── start              # Start the emulator stack
├── stop               # Stop the emulator stack
├── restart            # Restart the emulator stack
├── status             # Show status of all services
├── logs               # Show logs from services
├── policy             # Policy management
│   ├── validate       # Validate policy.yaml syntax
│   ├── init           # Initialize new policy file
│   ├── add-role       # Add a custom role
│   ├── add-binding    # Add an IAM binding
│   └── show           # Display current policy
├── test               # Testing utilities
│   └── permission     # Test a permission check
├── config             # Configuration management
│   ├── set            # Set a configuration value
│   ├── get            # Get configuration values
│   └── reset          # Reset to defaults
└── version            # Show version information
```

---

## Commands

### Stack Management

#### `gcp-emulator start`

Start the emulator stack using docker-compose.

**Usage:**
```bash
gcp-emulator start [flags]
```

**Flags:**
```
--mode string        IAM mode (off|permissive|strict) (default "permissive")
--detach, -d         Run in background (default true)
--pull               Pull latest images before starting
--profile string     Docker compose profile to use
```

**Examples:**
```bash
# Start with default settings (permissive mode)
gcp-emulator start

# Start in strict mode
gcp-emulator start --mode=strict

# Start and pull latest images
gcp-emulator start --pull

# Start with CI profile (strict mode services)
gcp-emulator start --profile=ci
```

**Output:**
```
✓ Pulling images...
✓ Starting IAM Emulator...
✓ Starting Secret Manager...
✓ Starting KMS...

Stack is ready!
  IAM:            http://localhost:8080
  Secret Manager: grpc://localhost:9090, http://localhost:8081
  KMS:            grpc://localhost:9091, http://localhost:8082

Run 'gcp-emulator logs' to view logs
Run 'gcp-emulator status' to check health
```

---

#### `gcp-emulator stop`

Stop the emulator stack.

**Usage:**
```bash
gcp-emulator stop [flags]
```

**Flags:**
```
--remove-volumes, -v    Remove volumes
```

**Examples:**
```bash
# Stop all services
gcp-emulator stop

# Stop and remove volumes
gcp-emulator stop -v
```

**Output:**
```
✓ Stopping IAM Emulator...
✓ Stopping Secret Manager...
✓ Stopping KMS...

Stack stopped
```

---

#### `gcp-emulator restart`

Restart the emulator stack or specific service.

**Usage:**
```bash
gcp-emulator restart [service] [flags]
```

**Flags:**
```
--mode string    IAM mode (off|permissive|strict)
```

**Examples:**
```bash
# Restart entire stack
gcp-emulator restart

# Restart only IAM emulator
gcp-emulator restart iam

# Restart in strict mode
gcp-emulator restart --mode=strict
```

**Output:**
```
✓ Restarting IAM Emulator...
✓ Restarting Secret Manager...
✓ Restarting KMS...

Stack restarted
```

---

#### `gcp-emulator status`

Show health status of all services.

**Usage:**
```bash
gcp-emulator status [flags]
```

**Flags:**
```
--watch, -w      Watch status (refresh every 2s)
--json           Output as JSON
```

**Examples:**
```bash
# Show status once
gcp-emulator status

# Watch status continuously
gcp-emulator status --watch

# Get JSON output (for scripting)
gcp-emulator status --json
```

**Output:**
```
Service          Status    Mode         Uptime    Ports
───────────────────────────────────────────────────────────
IAM Emulator     ✓ UP      -            2m30s     8080
Secret Manager   ✓ UP      permissive   2m25s     9090, 8081
KMS              ✓ UP      permissive   2m25s     9091, 8082

Health Checks:
  IAM:            ✓ http://localhost:8080/health (200 OK)
  Secret Manager: ✓ http://localhost:8081/health (200 OK)
  KMS:            ✓ http://localhost:8082/health (200 OK)
```

---

#### `gcp-emulator logs`

Show logs from services.

**Usage:**
```bash
gcp-emulator logs [service] [flags]
```

**Flags:**
```
--follow, -f     Follow log output
--tail int       Number of lines to show (default 50)
--since string   Show logs since timestamp (e.g. 2m, 1h)
```

**Examples:**
```bash
# Show logs from all services
gcp-emulator logs

# Show logs from IAM emulator only
gcp-emulator logs iam

# Follow logs in real-time
gcp-emulator logs --follow

# Show last 100 lines
gcp-emulator logs --tail=100

# Show logs from last 5 minutes
gcp-emulator logs --since=5m
```

**Output:**
```
iam-emulator         | [INFO] Loaded policy from /policy.yaml
iam-emulator         | [INFO] Server listening on :8080
secret-manager       | [INFO] Starting Secret Manager on :9090
secret-manager       | [INFO] IAM mode: permissive
kms                  | [INFO] Starting KMS on :9090
```

---

### Policy Management

#### `gcp-emulator policy validate`

Validate policy.yaml syntax and structure.

**Usage:**
```bash
gcp-emulator policy validate [file] [flags]
```

**Flags:**
```
--strict    Strict validation (check for unused roles)
```

**Examples:**
```bash
# Validate default policy.yaml
gcp-emulator policy validate

# Validate specific file
gcp-emulator policy validate custom-policy.yaml

# Strict validation
gcp-emulator policy validate --strict
```

**Output (success):**
```
✓ YAML syntax valid
✓ Schema valid
✓ 3 roles defined
✓ 2 groups defined
✓ 1 project configured

Policy is valid!
```

**Output (errors):**
```
✗ Validation failed

Errors:
  Line 12: Invalid permission format: "secret.manager.get" (should be "secretmanager.secrets.get")
  Line 24: Unknown role: "roles/custom.missing"
  Line 35: Duplicate binding for principal "user:alice@example.com"

Fix these issues and try again.
```

---

#### `gcp-emulator policy init`

Initialize a new policy file from template.

**Usage:**
```bash
gcp-emulator policy init [flags]
```

**Flags:**
```
--template string    Template to use (basic|advanced|ci) (default "basic")
--force, -f          Overwrite existing policy.yaml
--output string      Output file (default "policy.yaml")
```

**Examples:**
```bash
# Create basic policy
gcp-emulator policy init

# Create advanced policy with examples
gcp-emulator policy init --template=advanced

# Create CI-focused policy
gcp-emulator policy init --template=ci

# Force overwrite
gcp-emulator policy init --force
```

**Output:**
```
✓ Created policy.yaml with basic template

The policy includes:
  - Developer role with Secret Manager + KMS permissions
  - CI role with read-only permissions
  - Example groups and bindings

Edit policy.yaml to customize for your project.
```

---

#### `gcp-emulator policy add-role`

Add a custom role to policy.yaml.

**Usage:**
```bash
gcp-emulator policy add-role <role-name> [permissions...] [flags]
```

**Flags:**
```
--description string    Role description
--interactive, -i       Interactive mode (prompt for permissions)
```

**Examples:**
```bash
# Add role with specific permissions
gcp-emulator policy add-role roles/custom.reader \
  secretmanager.secrets.get \
  secretmanager.versions.access

# Interactive mode
gcp-emulator policy add-role roles/custom.writer --interactive

# With description
gcp-emulator policy add-role roles/custom.auditor \
  --description "Read-only access for auditing" \
  secretmanager.secrets.list \
  cloudkms.keyRings.list
```

**Output:**
```
✓ Added role: roles/custom.reader

Role includes permissions:
  - secretmanager.secrets.get
  - secretmanager.versions.access

To use this role, add a binding:
  gcp-emulator policy add-binding <project> roles/custom.reader user:alice@example.com
```

---

#### `gcp-emulator policy add-binding`

Add an IAM binding to a project.

**Usage:**
```bash
gcp-emulator policy add-binding <project> <role> <principal> [flags]
```

**Flags:**
```
--condition string    CEL condition expression
--title string        Condition title
--description string  Condition description
```

**Examples:**
```bash
# Simple binding
gcp-emulator policy add-binding test-project \
  roles/custom.developer \
  user:alice@example.com

# Binding with condition
gcp-emulator policy add-binding test-project \
  roles/custom.ciRunner \
  serviceAccount:ci@test.iam.gserviceaccount.com \
  --condition 'resource.name.startsWith("projects/test/secrets/prod-")' \
  --title "CI limited to prod secrets"

# Bind to group
gcp-emulator policy add-binding prod-project \
  roles/owner \
  group:developers
```

**Output:**
```
✓ Added binding to test-project

Role:      roles/custom.developer
Principal: user:alice@example.com

Run 'gcp-emulator restart' to apply changes.
```

---

#### `gcp-emulator policy show`

Display current policy in human-readable format.

**Usage:**
```bash
gcp-emulator policy show [flags]
```

**Flags:**
```
--format string    Output format (text|yaml|json) (default "text")
--project string   Filter by project
```

**Examples:**
```bash
# Show full policy
gcp-emulator policy show

# Show as YAML
gcp-emulator policy show --format=yaml

# Show specific project
gcp-emulator policy show --project=test-project
```

**Output:**
```
Roles:
  roles/custom.developer
    - secretmanager.secrets.create
    - secretmanager.secrets.get
    - cloudkms.cryptoKeys.encrypt

Groups:
  developers
    - user:alice@example.com
    - user:bob@example.com

Projects:
  test-project
    Binding 1:
      Role:      roles/custom.developer
      Members:   group:developers
      Condition: none
```

---

### Testing

#### `gcp-emulator test permission`

Test if a principal has a specific permission on a resource.

**Usage:**
```bash
gcp-emulator test permission <principal> <resource> <permission> [flags]
```

**Flags:**
```
--verbose, -v    Show detailed evaluation trace
```

**Examples:**
```bash
# Test permission
gcp-emulator test permission \
  user:alice@example.com \
  projects/test/secrets/db-password \
  secretmanager.secrets.get

# Verbose output
gcp-emulator test permission \
  serviceAccount:ci@test.iam.gserviceaccount.com \
  projects/test/secrets/prod-api-key \
  secretmanager.versions.access \
  --verbose
```

**Output (allowed):**
```
✓ ALLOWED

Principal:  user:alice@example.com
Resource:   projects/test/secrets/db-password
Permission: secretmanager.secrets.get

Granted by:
  Role:    roles/custom.developer
  Binding: projects/test-project (via group:developers)
```

**Output (denied):**
```
✗ DENIED

Principal:  user:charlie@example.com
Resource:   projects/test/secrets/db-password
Permission: secretmanager.secrets.get

Reason: No matching bindings found

Checked bindings:
  ✗ roles/custom.developer (principal not a member)
  ✗ roles/custom.ciRunner (condition not satisfied)
```

**Output (verbose):**
```
✓ ALLOWED

Evaluation trace:
  1. Principal: user:alice@example.com
  2. Resource:  projects/test/secrets/db-password
  3. Permission: secretmanager.secrets.get
  4. Checking project: test-project
     → Binding 1: roles/custom.developer
       → Members: group:developers
       → Group expansion: developers → [user:alice@example.com, user:bob@example.com]
       → Match: user:alice@example.com (via group)
       → Role permissions: [secretmanager.secrets.create, secretmanager.secrets.get, ...]
       → Permission match: secretmanager.secrets.get
       → Condition: none
       → Result: ALLOW

Final decision: ALLOW (via roles/custom.developer)
```

---

### Configuration

#### `gcp-emulator config set`

Set a configuration value.

**Usage:**
```bash
gcp-emulator config set <key> <value>
```

**Available keys:**
- `iam-mode`: Default IAM mode (off|permissive|strict)
- `pull-on-start`: Pull images before starting (true|false)
- `trace`: Enable IAM trace logging (true|false)
- `policy-file`: Path to policy.yaml (default: ./policy.yaml)

**Examples:**
```bash
# Set default IAM mode
gcp-emulator config set iam-mode strict

# Enable trace logging
gcp-emulator config set trace true

# Use custom policy file
gcp-emulator config set policy-file /etc/emulator/policy.yaml

# Auto-pull images
gcp-emulator config set pull-on-start true
```

**Output:**
```
✓ Configuration updated

iam-mode: strict

Restart the stack for changes to take effect:
  gcp-emulator restart
```

---

#### `gcp-emulator config get`

Get configuration values.

**Usage:**
```bash
gcp-emulator config get [key]
```

**Examples:**
```bash
# Show all configuration
gcp-emulator config get

# Get specific value
gcp-emulator config get iam-mode
```

**Output:**
```
Configuration:
  iam-mode:       permissive
  pull-on-start:  false
  trace:          false
  policy-file:    ./policy.yaml

Stored in: ~/.gcp-emulator/config.yaml
```

---

#### `gcp-emulator config reset`

Reset configuration to defaults.

**Usage:**
```bash
gcp-emulator config reset
```

**Output:**
```
✓ Configuration reset to defaults

Default values:
  iam-mode:       permissive
  pull-on-start:  false
  trace:          false
  policy-file:    ./policy.yaml
```

---

### Utility Commands

#### `gcp-emulator version`

Show version information.

**Usage:**
```bash
gcp-emulator version [flags]
```

**Flags:**
```
--short    Show only version number
```

**Examples:**
```bash
# Full version info
gcp-emulator version

# Short version
gcp-emulator version --short
```

**Output:**
```
gcp-emulator version v0.1.0

Components:
  IAM Emulator:      v0.5.0
  Secret Manager:    v1.2.0
  KMS:               v0.2.0
  gcp-emulator-auth: v0.1.1

Build:
  Commit:  75978ed
  Date:    2026-01-27T10:30:00Z
  Go:      go1.24.0
```

---

## Project Structure

```
gcp-emulator-control-plane/
├── cmd/
│   └── gcp-emulator/
│       └── main.go              # CLI entry point
├── internal/
│   ├── cli/
│   │   ├── root.go              # Root command
│   │   ├── start.go             # Start command
│   │   ├── stop.go              # Stop command
│   │   ├── restart.go           # Restart command
│   │   ├── status.go            # Status command
│   │   ├── logs.go              # Logs command
│   │   ├── policy.go            # Policy command group
│   │   ├── policy_validate.go  # Policy validation
│   │   ├── policy_init.go       # Policy initialization
│   │   ├── policy_add_role.go  # Add role
│   │   ├── policy_add_binding.go # Add binding
│   │   ├── policy_show.go       # Show policy
│   │   ├── test.go              # Test command group
│   │   ├── test_permission.go   # Permission testing
│   │   ├── config.go            # Config command group
│   │   └── version.go           # Version command
│   ├── docker/
│   │   ├── compose.go           # Docker compose wrapper
│   │   └── health.go            # Health checking
│   ├── policy/
│   │   ├── parser.go            # YAML parsing
│   │   ├── validator.go         # Policy validation
│   │   ├── modifier.go          # Policy modification
│   │   └── templates.go         # Policy templates
│   └── config/
│       ├── config.go            # Configuration management
│       └── defaults.go          # Default values
├── docker-compose.yml
├── policy.yaml
└── README.md
```

---

## Configuration File

**Location:** `~/.gcp-emulator/config.yaml`

**Format:**
```yaml
iam-mode: permissive
pull-on-start: false
trace: false
policy-file: ./policy.yaml
```

**Viper Configuration Management:**

Viper handles configuration precedence automatically:

1. **Command-line flags** (highest priority)
   ```bash
   gcp-emulator start --mode=strict
   ```

2. **Environment variables**
   ```bash
   export GCP_EMULATOR_IAM_MODE=strict
   export GCP_EMULATOR_TRACE=true
   gcp-emulator start
   ```

3. **Config file** (`~/.gcp-emulator/config.yaml`)
   ```yaml
   iam-mode: permissive
   trace: false
   ```

4. **Defaults** (lowest priority)
   ```go
   viper.SetDefault("iam-mode", "permissive")
   viper.SetDefault("trace", false)
   ```

**Implementation:**
```go
// internal/config/config.go
package config

import (
    "github.com/spf13/viper"
)

func Init() error {
    // Set config file name and type
    viper.SetConfigName("config")
    viper.SetConfigType("yaml")
    
    // Add config file search paths
    viper.AddConfigPath("$HOME/.gcp-emulator")
    viper.AddConfigPath(".")
    
    // Set defaults
    viper.SetDefault("iam-mode", "permissive")
    viper.SetDefault("pull-on-start", false)
    viper.SetDefault("trace", false)
    viper.SetDefault("policy-file", "./policy.yaml")
    
    // Bind environment variables with prefix
    viper.SetEnvPrefix("GCP_EMULATOR")
    viper.AutomaticEnv()
    
    // Read config file (ignore if not found)
    if err := viper.ReadInConfig(); err != nil {
        if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
            return err
        }
    }
    
    return nil
}

// Get retrieves a config value
func Get(key string) string {
    return viper.GetString(key)
}

// GetBool retrieves a boolean config value
func GetBool(key string) bool {
    return viper.GetBool(key)
}

// Set updates a config value
func Set(key string, value interface{}) error {
    viper.Set(key, value)
    return viper.WriteConfig()
}
```

**Cobra + Viper Integration:**
```go
// cmd/gcp-emulator/start.go
package main

import (
    "github.com/spf13/cobra"
    "github.com/spf13/viper"
)

var startCmd = &cobra.Command{
    Use:   "start",
    Short: "Start the emulator stack",
    Run: func(cmd *cobra.Command, args []string) {
        // Viper automatically resolves from flags, env, config, defaults
        mode := viper.GetString("iam-mode")
        trace := viper.GetBool("trace")
        pullOnStart := viper.GetBool("pull-on-start")
        
        // Use values...
    },
}

func init() {
    // Define flags
    startCmd.Flags().String("mode", "", "IAM mode (off|permissive|strict)")
    startCmd.Flags().Bool("pull", false, "Pull images before starting")
    startCmd.Flags().BoolP("detach", "d", true, "Run in background")
    
    // Bind flags to viper keys
    viper.BindPFlag("iam-mode", startCmd.Flags().Lookup("mode"))
    viper.BindPFlag("pull-on-start", startCmd.Flags().Lookup("pull"))
    viper.BindPFlag("detach", startCmd.Flags().Lookup("detach"))
}
```

**Environment Variable Naming:**

Viper automatically converts config keys to environment variables:
- Config key: `iam-mode` → Environment: `GCP_EMULATOR_IAM_MODE`
- Config key: `pull-on-start` → Environment: `GCP_EMULATOR_PULL_ON_START`
- Config key: `trace` → Environment: `GCP_EMULATOR_TRACE`

**Example: All three precedence levels:**
```bash
# Config file has: iam-mode: permissive
# Environment has: GCP_EMULATOR_IAM_MODE=strict
# Flag provided: --mode=off

# Result: flag wins (off)
gcp-emulator start --mode=off

# Without flag, env var wins
gcp-emulator start  # Uses strict from environment

# Without flag or env, config file wins
unset GCP_EMULATOR_IAM_MODE
gcp-emulator start  # Uses permissive from config file

# Without any, default wins
rm ~/.gcp-emulator/config.yaml
gcp-emulator start  # Uses permissive from defaults
```

---

## Dependencies

**Go modules:**
```go
require (
    github.com/spf13/cobra v1.8.0       // CLI framework
    github.com/spf13/viper v1.18.2      // Configuration management
    github.com/fatih/color v1.16.0      // Colored output
    gopkg.in/yaml.v3 v3.0.1             // YAML parsing
    github.com/google/cel-go v0.18.2    // CEL validation (optional)
)
```

**Why this stack:**
- **Cobra**: Industry standard CLI framework (kubectl, docker, gh use it)
- **Viper**: Perfect companion to Cobra, handles all config sources seamlessly
- **fatih/color**: Simple, cross-platform colored output
- **yaml.v3**: Direct policy file manipulation
- **cel-go**: Optional, for validating CEL conditions in policy

---

## Color Scheme

**Status indicators:**
- ✓ Green: Success, healthy, allowed
- ✗ Red: Error, stopped, denied
- ⚠ Yellow: Warning, starting, degraded
- → Cyan: Information, logs, actions

**Service status:**
- Green: UP
- Red: DOWN
- Yellow: STARTING

**Permission results:**
- Green: ALLOWED
- Red: DENIED

---

## Shell Completion

Generate completion scripts for various shells:

```bash
# Bash
gcp-emulator completion bash > /etc/bash_completion.d/gcp-emulator

# Zsh
gcp-emulator completion zsh > "${fpath[1]}/_gcp-emulator"

# Fish
gcp-emulator completion fish > ~/.config/fish/completions/gcp-emulator.fish
```

---

## Installation

**From source:**
```bash
cd gcp-emulator-control-plane
go install ./cmd/gcp-emulator
```

**From release:**
```bash
# Download from GitHub releases
curl -LO https://github.com/blackwell-systems/gcp-emulator-control-plane/releases/download/v0.1.0/gcp-emulator_linux_amd64
chmod +x gcp-emulator_linux_amd64
mv gcp-emulator_linux_amd64 /usr/local/bin/gcp-emulator
```

**Using Go install:**
```bash
go install github.com/blackwell-systems/gcp-emulator-control-plane/cmd/gcp-emulator@latest
```

---

## Future Enhancements

### Phase 2 Features
- [ ] `gcp-emulator ui` - Web UI for policy visualization
- [ ] `gcp-emulator export` - Export traces to file
- [ ] `gcp-emulator import` - Import policy from GCP project
- [ ] `gcp-emulator diff` - Compare two policy files
- [ ] `gcp-emulator explain` - Explain permission decision in detail

### Phase 3 Features
- [ ] Plugin system for custom emulators
- [ ] Policy testing framework (unit tests for policies)
- [ ] CI integration helpers
- [ ] Metrics and monitoring

---

## Design Principles

1. **Zero-config default:** `gcp-emulator start` should work immediately
2. **Progressive disclosure:** Simple commands for common tasks, flags for advanced usage
3. **Clear feedback:** Always show what's happening and what to do next
4. **Fail gracefully:** Helpful error messages with suggestions
5. **Scriptable:** Support JSON output and exit codes for automation
6. **Docker-aware:** Handle Docker errors gracefully, suggest fixes
7. **Policy-first:** Make policy management intuitive and safe

---

## Example Workflows

### First-time user
```bash
# Initialize policy
gcp-emulator policy init

# Start stack
gcp-emulator start

# Check status
gcp-emulator status

# Test permission
gcp-emulator test permission user:alice@example.com \
  projects/test/secrets/db-password \
  secretmanager.secrets.get
```

### Developer workflow
```bash
# Start in permissive mode
gcp-emulator start

# Add new role
gcp-emulator policy add-role roles/custom.backend \
  secretmanager.secrets.get \
  cloudkms.cryptoKeys.decrypt

# Apply changes
gcp-emulator restart

# Watch logs
gcp-emulator logs --follow
```

### CI/CD workflow
```bash
# Start in strict mode
gcp-emulator start --mode=strict

# Validate policy first
gcp-emulator policy validate --strict

# Run tests
go test ./...

# Check permission traces
gcp-emulator logs iam | grep DENY

# Stop
gcp-emulator stop
```

---

## Non-Goals

- **Not a GUI tool** (use `gcp-emulator ui` for that in future)
- **Not a replacement for docker-compose** (wraps it, doesn't replace it)
- **Not a deployment tool** (for local dev/CI only)
- **Not cloud-aware** (doesn't connect to real GCP)

---

## Success Metrics

CLI is successful if:
- New users can start stack in < 1 minute
- Common tasks require single command
- Error messages are actionable
- Documentation is rarely needed
- Users prefer CLI over docker-compose
