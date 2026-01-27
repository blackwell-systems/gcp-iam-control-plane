# Architecture

This document describes the system design, component interactions, and technical architecture of the GCP Emulator Control Plane.

---

## Table of Contents

1. [Repository Ecosystem](#repository-ecosystem)
2. [Complete System Architecture](#complete-system-architecture)
3. [High-Level Architecture](#high-level-architecture)
4. [Component Responsibilities](#component-responsibilities)
5. [CLI Architecture](#cli-architecture)
6. [Request Flow](#request-flow)
7. [Identity Propagation](#identity-propagation)
8. [Authorization Model](#authorization-model)
9. [Failure Modes](#failure-modes)
10. [Network Topology](#network-topology)
11. [Data Flow](#data-flow)
12. [Design Decisions](#design-decisions)
13. [Extension Points](#extension-points)

---

## Repository Ecosystem

The GCP Emulator Control Plane is composed of **5 repositories** working together:

```
┌────────────────────────────────────────────────────────────────────────────┐
│                     Repository Ecosystem Overview                          │
│                                                                            │
│  ┌──────────────────────────────────────────────────────────────────────┐ │
│  │  1. gcp-emulator-control-plane (THIS REPO)                           │ │
│  │     https://github.com/blackwell-systems/gcp-emulator-control-plane  │ │
│  │                                                                       │ │
│  │  Purpose: Orchestration layer for entire emulator ecosystem          │ │
│  │  Language: Go                                                         │ │
│  │  Provides:                                                            │ │
│  │    - gcp-emulator CLI (Cobra + Viper + fatih/color)                  │ │
│  │    - docker-compose.yml (service orchestration)                      │ │
│  │    - policy.yaml (single source of truth for IAM)                    │ │
│  │    - Policy packs (secretmanager.yaml, kms.yaml, ci.yaml)            │ │
│  │    - Integration contract documentation                              │ │
│  │    - End-to-end examples (Go SDK + curl)                             │ │
│  │                                                                       │ │
│  │  Key Components:                                                      │ │
│  │    - cmd/gcp-emulator/main.go - CLI entry point                      │ │
│  │    - internal/cli/ - Cobra command implementations                   │ │
│  │    - internal/config/ - Viper configuration (disciplined pattern)    │ │
│  │    - internal/docker/ - Docker compose wrapper                       │ │
│  │    - internal/policy/ - Policy parser and validator                  │ │
│  │    - examples/ - Reference implementations                           │ │
│  └──────────────────────────────────────────────────────────────────────┘ │
│                                    │                                      │
│                                    │ orchestrates via docker-compose      │
│                                    ▼                                      │
│  ┌──────────────────────────────────────────────────────────────────────┐ │
│  │  2. gcp-iam-emulator (CONTROL PLANE)                                 │ │
│  │     https://github.com/blackwell-systems/gcp-iam-emulator            │ │
│  │                                                                       │ │
│  │  Purpose: Authorization engine for all emulators                     │ │
│  │  Language: Go                                                         │ │
│  │  Provides:                                                            │ │
│  │    - IAM Policy API (gRPC + REST)                                    │ │
│  │    - TestIamPermissions, SetIamPolicy, GetIamPolicy                  │ │
│  │    - Role expansion (custom roles + built-in roles)                  │ │
│  │    - Group resolution (1 level of nesting)                           │ │
│  │    - CEL condition evaluation (resource-based access control)        │ │
│  │    - Policy schema v3 support                                        │ │
│  │    - Hot reload (--watch flag)                                       │ │
│  │    - Enhanced trace mode (JSON output, metrics)                      │ │
│  │                                                                       │ │
│  │  Key Components:                                                      │ │
│  │    - cmd/server/main.go - IAM server binary                          │ │
│  │    - internal/policy/ - Policy loading and parsing                   │ │
│  │    - internal/engine/ - Permission evaluation logic                  │ │
│  │    - internal/roles/ - Built-in role definitions                     │ │
│  │    - internal/conditions/ - CEL expression evaluator                 │ │
│  │                                                                       │ │
│  │  Docker Image: ghcr.io/blackwell-systems/gcp-iam-emulator:latest    │ │
│  │  Default Port: 8080 (gRPC)                                           │ │
│  └──────────────────────────────────────────────────────────────────────┘ │
│                                    ▲                                      │
│                                    │ TestIamPermissions(principal,        │
│                                    │   resource, permission)              │
│                                    │                                      │
│  ┌─────────────────────────────────┴──────────────────────────────────┐  │
│  │  3. gcp-secret-manager-emulator (DATA PLANE)                        │  │
│  │     https://github.com/blackwell-systems/gcp-secret-manager-emulator│  │
│  │                                                                      │  │
│  │  Purpose: Secret Manager CRUD operations with IAM enforcement       │  │
│  │  Language: Go                                                        │  │
│  │  Provides:                                                           │  │
│  │    - Secret Manager API (gRPC + REST)                               │  │
│  │    - CreateSecret, GetSecret, UpdateSecret, DeleteSecret            │  │
│  │    - AddSecretVersion, AccessSecretVersion, ListSecretVersions      │  │
│  │    - EnableSecretVersion, DisableSecretVersion, DestroyVersion      │  │
│  │    - In-memory storage (hermetic, no persistence)                   │  │
│  │    - IAM mode support (off/permissive/strict)                       │  │
│  │    - 90.8% test coverage                                            │  │
│  │                                                                      │  │
│  │  Key Components:                                                     │  │
│  │    - cmd/server/main.go - gRPC-only server                          │  │
│  │    - cmd/server-rest/main.go - REST-only server                     │  │
│  │    - cmd/server-dual/main.go - Dual protocol server                 │  │
│  │    - internal/server/ - gRPC service implementation                 │  │
│  │    - internal/storage/ - In-memory secret storage                   │  │
│  │    - Uses: gcp-emulator-auth (principal extraction, IAM client)     │  │
│  │                                                                      │  │
│  │  Docker Image: ghcr.io/blackwell-systems/gcp-secret-manager-emulator│  │
│  │  Default Ports: 9090 (gRPC), 8081 (HTTP)                            │  │
│  │  API Coverage: 11 of 12 methods (92%)                               │  │
│  └──────────────────────────────────────────────────────────────────────┘  │
│                                                                            │
│  ┌──────────────────────────────────────────────────────────────────────┐ │
│  │  4. gcp-kms-emulator (DATA PLANE)                                    │ │
│  │     https://github.com/blackwell-systems/gcp-kms-emulator            │ │
│  │                                                                       │ │
│  │  Purpose: KMS cryptographic operations with IAM enforcement          │ │
│  │  Language: Go                                                         │ │
│  │  Provides:                                                            │ │
│  │    - KMS API (gRPC + REST)                                           │ │
│  │    - CreateKeyRing, GetKeyRing, ListKeyRings                         │ │
│  │    - CreateCryptoKey, GetCryptoKey, ListCryptoKeys, UpdateCryptoKey  │ │
│  │    - CreateCryptoKeyVersion, UpdateCryptoKeyPrimaryVersion           │ │
│  │    - Encrypt, Decrypt (AES-256-GCM)                                  │ │
│  │    - Key version lifecycle (enable, disable, destroy)                │ │
│  │    - In-memory storage (hermetic, no persistence)                    │ │
│  │    - IAM mode support (off/permissive/strict)                        │ │
│  │                                                                       │ │
│  │  Key Components:                                                      │ │
│  │    - cmd/server/main.go - gRPC-only server                           │ │
│  │    - cmd/server-rest/main.go - REST-only server                      │ │
│  │    - cmd/server-dual/main.go - Dual protocol server                  │ │
│  │    - internal/server/ - gRPC service implementation                  │ │
│  │    - internal/storage/ - In-memory key/keyring storage               │ │
│  │    - internal/crypto/ - AES-256-GCM encryption                       │ │
│  │    - Uses: gcp-emulator-auth (principal extraction, IAM client)      │ │
│  │                                                                       │ │
│  │  Docker Image: ghcr.io/blackwell-systems/gcp-kms-emulator:latest    │ │
│  │  Default Ports: 9091 (gRPC), 8082 (HTTP)                             │ │
│  │  API Coverage: 14 of ~26 methods (54% - complete key mgmt)          │ │
│  └──────────────────────────────────────────────────────────────────────┘ │
│                                                                            │
│  ┌──────────────────────────────────────────────────────────────────────┐ │
│  │  5. gcp-emulator-auth (SHARED LIBRARY)                               │ │
│  │     https://github.com/blackwell-systems/gcp-emulator-auth           │ │
│  │                                                                       │ │
│  │  Purpose: Shared auth/authz logic for all data plane emulators       │ │
│  │  Language: Go                                                         │ │
│  │  Provides:                                                            │ │
│  │    - Principal extraction (gRPC metadata + HTTP headers)             │ │
│  │    - Environment config parsing (IAM_MODE, IAM_HOST)                 │ │
│  │    - IAM client wrapper with timeout and mode handling               │ │
│  │    - Error classification (connectivity vs config errors)            │ │
│  │    - Consistent behavior across all emulators                        │ │
│  │                                                                       │ │
│  │  Key Functions:                                                       │ │
│  │    - ExtractPrincipalFromContext(ctx) string                         │ │
│  │    - ExtractPrincipalFromRequest(r *http.Request) string             │ │
│  │    - LoadFromEnv() *Config                                           │ │
│  │    - NewClient(host string, mode Mode) (*Client, error)              │ │
│  │    - CheckPermission(ctx, principal, resource, perm) (bool, error)   │ │
│  │                                                                       │ │
│  │  Imported by: gcp-secret-manager-emulator, gcp-kms-emulator          │ │
│  │  Why: Prevents code drift, ensures consistent IAM integration        │ │
│  └──────────────────────────────────────────────────────────────────────┘ │
└────────────────────────────────────────────────────────────────────────────┘
```

### Repository Dependency Graph

```
gcp-emulator-control-plane (orchestration)
    │
    ├─ docker-compose.yml orchestrates ──────┐
    │                                         │
    ▼                                         ▼
gcp-iam-emulator                    gcp-secret-manager-emulator
(control plane)                     (data plane)
    ▲                                         │
    │                                         │ import
    │                                         ▼
    │                               gcp-emulator-auth
    │                               (shared library)
    │                                         ▲
    │                                         │ import
    │                                         │
    └─────────────────────────────────────────┤
                                              │
                                    gcp-kms-emulator
                                    (data plane)
```

### Repository Relationships

**Control Plane (orchestration repo):**
- `gcp-emulator-control-plane` - CLI, docker-compose, policy.yaml, documentation

**Control Plane (authorization engine):**
- `gcp-iam-emulator` - Standalone authorization service, no dependencies on data plane

**Data Plane (service emulators):**
- `gcp-secret-manager-emulator` - Imports `gcp-emulator-auth`, calls `gcp-iam-emulator` gRPC
- `gcp-kms-emulator` - Imports `gcp-emulator-auth`, calls `gcp-iam-emulator` gRPC

**Shared Library:**
- `gcp-emulator-auth` - Imported by all data plane emulators, no dependencies on other repos

### Key Design Principles

**Separation of Concerns:**
- Control plane repo (this repo) = orchestration, CLI, documentation
- IAM repo = authorization logic only
- Data plane repos = CRUD operations + IAM integration
- Shared library = common auth code (DRY principle)

**No Circular Dependencies:**
- IAM emulator is standalone (no knowledge of data plane)
- Data plane emulators depend on shared library only
- Shared library has no dependencies on other emulators
- Control plane repo orchestrates via docker-compose (runtime, not compile-time)

**Versioning Strategy:**
- Each repo has independent versioning
- Control plane `docker-compose.yml` pins specific versions
- Shared library uses semantic versioning
- Breaking changes require coordination across repos

### Docker Image Publishing

All repos publish Docker images to GitHub Container Registry (GHCR):

```
ghcr.io/blackwell-systems/gcp-iam-emulator:latest
ghcr.io/blackwell-systems/gcp-iam-emulator:v1.0.0

ghcr.io/blackwell-systems/gcp-secret-manager-emulator:latest
ghcr.io/blackwell-systems/gcp-secret-manager-emulator:v1.2.0

ghcr.io/blackwell-systems/gcp-kms-emulator:latest
ghcr.io/blackwell-systems/gcp-kms-emulator:v0.2.0
```

**Publishing flow:**
1. GitHub Actions on each repo builds multi-arch images (linux/amd64, linux/arm64)
2. Images pushed to GHCR on tag push
3. Control plane `docker-compose.yml` references these published images
4. Users run `gcp-emulator start` → CLI pulls images → stack runs

---

## Complete System Architecture

The GCP Emulator Control Plane consists of multiple layers working together:

```
┌────────────────────────────────────────────────────────────────┐
│                        User Layer                              │
│  ┌──────────────────┐        ┌─────────────────────────────┐   │
│  │  gcp-emulator    │        │  Client Applications        │   │
│  │  CLI Tool        │        │  (Go SDK, curl, scripts)    │   │
│  │                  │        │                             │   │
│  │  - Cobra         │        └─────────────┬───────────────┘   │
│  │  - Viper         │                      │                   │
│  │  - fatih/color   │                      │                   │
│  └────────┬─────────┘                      │                   │
│           │                                │                   │
└───────────┼────────────────────────────────┼───────────────────┘
            │ docker-compose                 │ gRPC/HTTP
            │ commands                       │ with X-Emulator-Principal
            ↓                                ↓
┌────────────────────────────────────────────────────────────────┐
│                    Orchestration Layer                         │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │         Docker Compose (docker-compose.yml)              │  │
│  │  - Service definitions                                   │  │
│  │  - Health checks                                         │  │
│  │  - Network configuration                                 │  │
│  │  - Environment variable injection                        │  │
│  └──────────────────────────────────────────────────────────┘  │
└────────────────────────────────────────────────────────────────┘
            │
            │ Container orchestration
            ↓
┌────────────────────────────────────────────────────────────────┐
│                    Container Layer                             │
│  ┌─────────────┐    ┌──────────────────┐    ┌─────────────┐   │
│  │ IAM         │    │ Secret Manager   │    │ KMS         │   │
│  │ Container   │    │ Container        │    │ Container   │   │
│  │             │    │                  │    │             │   │
│  │ Port: 8080  │    │ gRPC: 9090       │    │ gRPC: 9091  │   │
│  │             │    │ HTTP: 8081       │    │ HTTP: 8082  │   │
│  └─────────────┘    └──────────────────┘    └─────────────┘   │
└────────────────────────────────────────────────────────────────┘
            │
            │ gRPC calls
            ↓
┌────────────────────────────────────────────────────────────────┐
│                    Service Layer                               │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │            IAM Emulator (Control Plane)                  │  │
│  │  - Policy evaluation from policy.yaml                    │  │
│  │  - Role expansion                                        │  │
│  │  - Group resolution                                      │  │
│  │  - CEL condition evaluation                              │  │
│  └──────────────────────────────────────────────────────────┘  │
│         ▲                                                      │
│         │ TestIamPermissions(principal, resource, permission) │
│         │                                                      │
│  ┌──────┴────────────────┐       ┌──────────────────────────┐ │
│  │ Secret Manager        │       │ KMS Emulator             │ │
│  │ Emulator              │       │                          │ │
│  │ - CRUD operations     │       │ - Key management         │ │
│  │ - Permission checks   │       │ - Encrypt/Decrypt        │ │
│  │ - Storage (in-memory) │       │ - Permission checks      │ │
│  │ - gcp-emulator-auth   │       │ - gcp-emulator-auth      │ │
│  └───────────────────────┘       └──────────────────────────┘ │
└────────────────────────────────────────────────────────────────┘
```

### Layer Responsibilities

**User Layer:**
- CLI tool for managing stack (`gcp-emulator`)
- Client applications using emulators
- Principal injection via headers/metadata

**Orchestration Layer:**
- Docker Compose defines service topology
- Health checks ensure proper startup order
- Network configuration for service discovery
- Environment variable propagation

**Container Layer:**
- Pre-built Docker images from GHCR
- Port mapping to host
- Volume mounts for policy.yaml
- Independent lifecycle management

**Service Layer:**
- Control plane: IAM policy evaluation
- Data plane: CRUD operations with permission checks
- Shared library: gcp-emulator-auth prevents code drift

---

## CLI Architecture

The `gcp-emulator` CLI provides a unified interface for managing the entire stack without requiring docker-compose knowledge.

### CLI Component Structure

```
┌──────────────────────────────────────────────────────────────┐
│                     CLI Architecture                         │
│                                                              │
│  ┌────────────────────────────────────────────────────────┐ │
│  │           cmd/gcp-emulator/main.go                     │ │
│  │  - Entry point                                         │ │
│  │  - Version injection                                   │ │
│  │  - Config initialization                               │ │
│  └─────────────────────┬────────────────────────────────── │
│                        │                                    │
│  ┌─────────────────────▼──────────────────────────────────┐ │
│  │         internal/cli/ (Cobra Commands)                │ │
│  │                                                        │ │
│  │  ┌──────────────┐  ┌────────────┐  ┌──────────────┐  │ │
│  │  │ root.go      │  │ start.go   │  │ status.go    │  │ │
│  │  │ - Help       │  │ - Start    │  │ - Health     │  │ │
│  │  │ - Version    │  │ - Pull     │  │ - Display    │  │ │
│  │  └──────────────┘  └────────────┘  └──────────────┘  │ │
│  │                                                        │ │
│  │  ┌──────────────┐  ┌────────────┐  ┌──────────────┐  │ │
│  │  │ policy.go    │  │ config.go  │  │ logs.go      │  │ │
│  │  │ - Validate   │  │ - Get      │  │ - Follow     │  │ │
│  │  │ - Init       │  │ - Set      │  │ - Filter     │  │ │
│  │  └──────────────┘  └────────────┘  └──────────────┘  │ │
│  └────────────────────┬───────────────────────────────── │
│                       │                                   │
│  ┌────────────────────▼──────────────────────────────────┐ │
│  │      internal/config/ (Viper Integration)            │ │
│  │                                                       │ │
│  │  ┌─────────────────────────────────────────────────┐ │ │
│  │  │ Config struct (explicit configuration)          │ │ │
│  │  │ - IAMMode     string                             │ │ │
│  │  │ - Trace       bool                               │ │ │
│  │  │ - PolicyFile  string                             │ │ │
│  │  │ - Ports       PortConfig                         │ │ │
│  │  └─────────────────────────────────────────────────┘ │ │
│  │                                                       │ │
│  │  ┌─────────────────────────────────────────────────┐ │ │
│  │  │ Configuration Precedence (Viper)                │ │ │
│  │  │ 1. Command-line flags                           │ │ │
│  │  │ 2. Environment variables (GCP_EMULATOR_*)       │ │ │
│  │  │ 3. Config file (~/.gcp-emulator/config.yaml)    │ │ │
│  │  │ 4. Defaults                                     │ │ │
│  │  └─────────────────────────────────────────────────┘ │ │
│  └───────────────────┬───────────────────────────────── │
│                      │                                   │
│  ┌───────────────────▼──────────────────────────────────┐ │
│  │    internal/docker/ (Docker Compose Wrapper)        │ │
│  │  - Start(cfg)                                       │ │
│  │  - Stop()                                           │ │
│  │  - Status() → ServiceStatus                         │ │
│  │  - Pull()                                           │ │
│  │  - Logs()                                           │ │
│  └─────────────────────────────────────────────────────┘ │
│                      │                                   │
│  ┌───────────────────▼──────────────────────────────────┐ │
│  │    internal/policy/ (Policy Parser & Validator)     │ │
│  │  - Parse(file) → Policy                             │ │
│  │  - Validate(policy) → ValidationResult              │ │
│  │  - Init(template) → Policy                          │ │
│  └─────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────┘
           │                                │
           │ docker-compose commands        │ policy.yaml
           ▼                                ▼
    Docker Engine                    Filesystem
```

### Design Principles

**1. Disciplined Viper Pattern**

The CLI follows the "disciplined Viper" pattern from CLI_VIPER_PATTERN.md:

```
Viper = configuration resolution engine (stays in internal/config/)
Config struct = explicit configuration (passed to business logic)
Business logic = no viper imports (clean, testable)
```

**Example flow:**
```go
// internal/config/config.go
type Config struct {
    IAMMode string
    Ports   PortConfig
}

func Load() (*Config, error) {
    cfg := &Config{
        IAMMode: viper.GetString("iam-mode"),
        Ports: PortConfig{
            IAM: viper.GetInt("port-iam"),
        },
    }
    return cfg, cfg.Validate()
}

// internal/cli/start.go
func RunE(cmd *cobra.Command, args []string) error {
    cfg, _ := config.Load()  // Explicit config
    return docker.Start(cfg)  // Pass explicit config
}

// internal/docker/compose.go
func Start(cfg *config.Config) error {
    // No viper imports! Uses explicit config
    env := []string{
        fmt.Sprintf("IAM_MODE=%s", cfg.IAMMode),
    }
    // ...
}
```

**2. Colored Output for UX**

Using `fatih/color` for terminal output:

```
✓ (green)  - Success
✗ (red)    - Error
⚠ (yellow) - Warning
→ (cyan)   - Info/Progress
```

**3. Configuration Precedence**

Users can configure the CLI in multiple ways (highest precedence first):

1. **Command-line flags:**
   ```bash
   gcp-emulator start --mode=strict --pull
   ```

2. **Environment variables:**
   ```bash
   export GCP_EMULATOR_IAM_MODE=permissive
   export GCP_EMULATOR_PORT_IAM=8080
   ```

3. **Config file (`~/.gcp-emulator/config.yaml`):**
   ```yaml
   iam-mode: permissive
   port-iam: 8080
   policy-file: ./policy.yaml
   ```

4. **Defaults:**
   ```go
   viper.SetDefault("iam-mode", "off")
   viper.SetDefault("port-iam", 8080)
   ```

### Command Flow Examples

**Starting the stack:**

```
User: gcp-emulator start --mode=permissive

1. Cobra parses flags
   ├─ mode flag → "permissive"
   
2. Viper resolves configuration
   ├─ Bind flag to viper key "iam-mode"
   ├─ Check precedence: flag > env > config > default
   └─ Result: iam-mode = "permissive"

3. Config.Load() creates explicit struct
   ├─ cfg := &Config{IAMMode: "permissive", ...}
   └─ cfg.Validate() checks constraints

4. docker.Start(cfg) executes
   ├─ Build env vars from cfg (no viper!)
   ├─ exec.Command("docker-compose", "up", "-d")
   └─ Return success/error

5. CLI prints colored output
   ├─ color.Cyan("→ Starting stack...")
   └─ color.Green("✓ Stack started successfully")
```

**Validating policy:**

```
User: gcp-emulator policy validate

1. Cobra routes to policy.validateCmd

2. policy.Parse(file) reads YAML
   ├─ yaml.Unmarshal into Policy struct
   └─ Return parsed policy

3. policy.Validate(policy) checks constraints
   ├─ Check role names start with "roles/"
   ├─ Check permission format (service.resource.verb)
   ├─ Check custom roles are defined
   ├─ Check bindings reference valid roles
   └─ Return ValidationResult{Valid, Errors}

4. CLI displays results
   ├─ If valid: color.Green("✓ Policy is valid")
   └─ If errors: color.Red("✗ Validation failed") + list errors
```

### Why This Architecture?

**Testability:**
- Explicit Config struct → easy to test with mock configs
- No global Viper state in business logic → isolated tests
- Docker wrapper → can mock exec.Command

**Maintainability:**
- Clear separation: CLI (Cobra) → Config (Viper) → Business logic
- Single source of truth for configuration precedence
- Easy to add new commands (just add to internal/cli/)

**User Experience:**
- Colored output improves readability
- Consistent flag naming across commands
- Configuration flexibility (flags/env/file/defaults)
- Single binary, no docker-compose knowledge required

**Production-Grade:**
- Same stack as kubectl, docker CLI, helm (Cobra + Viper)
- Well-tested libraries with large communities
- Cross-platform support (Windows/Linux/macOS)

---

## High-Level Architecture

The GCP Emulator Control Plane uses a **control plane + data plane** architecture modeled after real GCP:

```
┌─────────────────────────────────────────────────────────┐
│                    Control Plane                        │
│  ┌──────────────────────────────────────────────────┐   │
│  │            IAM Emulator (policy.yaml)            │   │
│  │  - Policy evaluation                             │   │
│  │  - Role expansion                                │   │
│  │  - Condition evaluation (CEL)                    │   │
│  │  - Group membership resolution                   │   │
│  └──────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
                         ▲
                         │ CheckPermission(principal, resource, permission)
                         │
┌────────────────────────┴────────────────────────────────┐
│                     Data Plane                          │
│  ┌─────────────────────┐    ┌─────────────────────┐    │
│  │ Secret Manager      │    │  KMS Emulator       │    │
│  │ Emulator            │    │                     │    │
│  │ - CRUD operations   │    │  - Key management   │    │
│  │ - Permission checks │    │  - Encrypt/Decrypt  │    │
│  │ - Resource storage  │    │  - Permission checks│    │
│  └─────────────────────┘    └─────────────────────┘    │
└─────────────────────────────────────────────────────────┘
                         ▲
                         │ X-Emulator-Principal header
                         │
                   ┌─────┴─────┐
                   │   Client  │
                   └───────────┘
```

### Key Characteristics

- **Separation of concerns**: Control plane handles authorization, data plane handles operations
- **Centralized policy**: Single `policy.yaml` drives all authorization decisions
- **Stateless data plane**: Emulators don't store policy, they delegate to IAM emulator
- **Consistent identity channel**: Principal injection works the same across all services

---

## Component Responsibilities

### IAM Emulator (Control Plane)

**Purpose:** Enforce authorization policy across all data plane services

**Responsibilities:**
- Load and parse `policy.yaml`
- Evaluate `TestIamPermissions` requests
- Expand roles into permission sets
- Resolve group memberships
- Evaluate CEL condition expressions
- Return allow/deny decisions

**Does NOT:**
- Store or manage data plane resources
- Execute business logic
- Handle data plane CRUD operations
- Maintain resource state

**Interfaces:**
- gRPC API: `google.iam.v1.IAMPolicy` service
- HTTP endpoint: `/health` for readiness checks
- Config: `policy.yaml` file mount

### Secret Manager Emulator (Data Plane)

**Purpose:** Emulate GCP Secret Manager CRUD operations with IAM enforcement

**Responsibilities:**
- Store secrets and versions in-memory
- Validate request parameters
- Check permissions via IAM emulator (when IAM_MODE enabled)
- Execute secret operations (create, read, update, delete)
- Return appropriate error codes

**Does NOT:**
- Store or evaluate IAM policy
- Make authorization decisions (delegates to IAM emulator)
- Persist data across restarts

**Interfaces:**
- gRPC API: `google.cloud.secretmanager.v1.SecretManagerService`
- HTTP API: REST gateway at `/v1/projects/{project}/secrets`
- IAM integration: Calls IAM emulator's `TestIamPermissions`

### KMS Emulator (Data Plane)

**Purpose:** Emulate GCP KMS cryptographic operations with IAM enforcement

**Responsibilities:**
- Manage key rings and crypto keys in-memory
- Perform encrypt/decrypt operations
- Check permissions via IAM emulator (when IAM_MODE enabled)
- Execute KMS operations (create keys, encrypt, decrypt)
- Return appropriate error codes

**Does NOT:**
- Store or evaluate IAM policy
- Use real cryptographic algorithms (uses simple XOR for testing)
- Persist keys across restarts

**Interfaces:**
- gRPC API: `google.cloud.kms.v1.KeyManagementService`
- HTTP API: REST gateway at `/v1/projects/{project}/locations/{location}/keyRings`
- IAM integration: Calls IAM emulator's `TestIamPermissions`

---

## Request Flow

### Standard Operation Flow

```
1. Client                     2. Data Plane               3. Control Plane
   │                             │                           │
   ├─ POST /secrets              │                           │
   │  X-Emulator-Principal:      │                           │
   │  user:alice@example.com     │                           │
   │                             │                           │
   │────────────────────────────>│                           │
   │                             │                           │
   │                             ├─ Extract principal        │
   │                             ├─ Validate request         │
   │                             ├─ Normalize resource       │
   │                             │                           │
   │                             ├─ TestIamPermissions       │
   │                             │  principal=alice          │
   │                             │  resource=projects/test   │
   │                             │  permission=secrets.create│
   │                             │                           │
   │                             │──────────────────────────>│
   │                             │                           │
   │                             │                           ├─ Load policy
   │                             │                           ├─ Expand roles
   │                             │                           ├─ Check bindings
   │                             │                           ├─ Evaluate conditions
   │                             │                           │
   │                             │<──────────────────────────┤
   │                             │  permissions=[            │
   │                             │    "secrets.create"       │
   │                             │  ]                        │
   │                             │                           │
   │                             ├─ Execute operation        │
   │                             ├─ Store resource           │
   │                             │                           │
   │<────────────────────────────┤                           │
   │  200 OK                     │                           │
   │  {secret}                   │                           │
```

### Permission Denied Flow

```
1. Client                     2. Data Plane               3. Control Plane
   │                             │                           │
   ├─ GET /secrets/prod-key      │                           │
   │  X-Emulator-Principal:      │                           │
   │  user:charlie@example.com   │                           │
   │                             │                           │
   │────────────────────────────>│                           │
   │                             │                           │
   │                             ├─ Extract principal        │
   │                             ├─ Validate request         │
   │                             │                           │
   │                             ├─ TestIamPermissions       │
   │                             │  principal=charlie        │
   │                             │  resource=projects/.../prod-key
   │                             │  permission=secrets.get   │
   │                             │                           │
   │                             │──────────────────────────>│
   │                             │                           │
   │                             │                           ├─ Check bindings
   │                             │                           ├─ No match found
   │                             │                           │
   │                             │<──────────────────────────┤
   │                             │  permissions=[]           │
   │                             │                           │
   │                             ├─ STOP (denied)            │
   │                             │                           │
   │<────────────────────────────┤                           │
   │  403 Forbidden              │                           │
   │  Permission denied          │                           │
```

---

## Identity Propagation

### Inbound: Client → Data Plane

Principal identity flows from client to data plane emulator via **standard headers**:

**gRPC:**
```
Metadata: x-emulator-principal: user:alice@example.com
```

**HTTP:**
```
Header: X-Emulator-Principal: user:alice@example.com
```

### Extraction Pattern

Data plane emulators extract the principal using `gcp-emulator-auth` library:

```go
import emulatorauth "github.com/blackwell-systems/gcp-emulator-auth"

principal := emulatorauth.ExtractPrincipalFromContext(ctx)
// Returns: "user:alice@example.com"
```

### Outbound: Data Plane → Control Plane

When calling IAM emulator, the data plane **propagates** the principal via metadata:

```go
// Inject principal into outgoing context
ctx = metadata.AppendToOutgoingContext(ctx, "x-emulator-principal", principal)

// Call IAM emulator
resp, err := iamClient.TestIamPermissions(ctx, &iampb.TestIamPermissionsRequest{
    Resource:    "projects/test-project/secrets/db-password",
    Permissions: []string{"secretmanager.secrets.get"},
})
```

### Why Metadata, Not Request Body?

**Design rationale:**
1. **Matches real GCP behavior**: GCP API requests don't include identity in the body
2. **Separation of concerns**: Identity is control plane concern, request is data plane
3. **Transparent forwarding**: Data plane can forward identity without parsing it
4. **Test realism**: Tests using real GCP SDK clients work without modification

---

## Authorization Model

### Policy Structure

```yaml
roles:                        # Role definitions
  roles/custom.developer:
    permissions:
      - secretmanager.secrets.create
      - secretmanager.secrets.get

groups:                       # Group membership
  developers:
    members:
      - user:alice@example.com

projects:                     # Resource hierarchy
  test-project:
    bindings:                 # IAM bindings
      - role: roles/custom.developer
        members:
          - group:developers
        condition:            # Optional CEL expression
          expression: 'resource.name.startsWith("projects/test-project/secrets/dev-")'
```

### Permission Check Algorithm

```
1. Extract principal from request
   → "user:alice@example.com"

2. Normalize resource path
   → "projects/test-project/secrets/db-password"

3. Call IAM emulator TestIamPermissions
   → CheckPermission(principal, resource, permission)

4. IAM emulator evaluates:
   a. Find all bindings for resource (project-level)
   b. Expand groups → alice is in "developers"
   c. Check if any binding grants required permission
   d. Evaluate conditions (if present)
   e. Return list of granted permissions

5. Data plane checks result:
   - If permission in result → ALLOW
   - If permission not in result → DENY (403)
```

### Condition Evaluation

Conditions use **Common Expression Language (CEL)**:

```yaml
condition:
  expression: 'resource.name.startsWith("projects/test-project/secrets/prod-")'
  title: "Restrict to production secrets"
```

**Evaluation context:**
- `resource.name`: Full resource path being accessed
- `request.time`: Timestamp of request
- Custom variables (future)

**CEL operators:**
- String: `startsWith()`, `endsWith()`, `contains()`, `matches()`
- Logical: `&&`, `||`, `!`
- Comparison: `==`, `!=`, `<`, `>`, `<=`, `>=`

---

## Failure Modes

### IAM Mode Behavior Matrix

| Scenario | IAM_MODE=off | IAM_MODE=permissive | IAM_MODE=strict |
|----------|--------------|---------------------|-----------------|
| IAM emulator healthy | No check (allow) | Check permission | Check permission |
| IAM emulator down | No check (allow) | Allow (fail-open) | Deny (fail-closed) |
| IAM returns error | No check (allow) | Allow (fail-open) | Deny (fail-closed) |
| No principal header | No check (allow) | Deny | Deny |
| Permission denied | No check (allow) | Deny | Deny |

### Error Propagation

```
Data Plane Error → Client Error Code

Permission denied     → 403 Forbidden (PermissionDenied)
IAM unavailable       → 500 Internal (Internal) [strict mode only]
Invalid request       → 400 Bad Request (InvalidArgument)
Resource not found    → 404 Not Found (NotFound)
```

### Recovery Strategies

**IAM emulator restart:**
- Data plane maintains connection pool
- Automatic reconnection on next request
- No data loss (policy in config file)

**Data plane restart:**
- In-memory data lost (emulator design)
- IAM state preserved (stateless data plane)
- Clients retry with standard backoff

**Network partition:**
- Permissive mode: Operations continue (fail-open)
- Strict mode: Operations blocked (fail-closed)
- Health checks detect partition

---

## Network Topology

### Docker Compose Deployment

```
┌─────────────────────────────────────────────────┐
│             Docker Network (bridge)             │
│                                                 │
│  ┌──────────────┐                               │
│  │ IAM Emulator │                               │
│  │ iam:8080     │◄──────────────┐               │
│  └──────────────┘               │               │
│         ▲                       │               │
│         │ CheckPermission       │               │
│         │                       │               │
│  ┌──────┴───────────┐    ┌──────┴──────────┐   │
│  │ Secret Manager   │    │  KMS Emulator   │   │
│  │ secret-mgr:9090  │    │  kms:9090       │   │
│  │ secret-mgr:8080  │    │  kms:8080       │   │
│  └──────────────────┘    └─────────────────┘   │
│         ▲                       ▲               │
│         │                       │               │
└─────────┼───────────────────────┼───────────────┘
          │                       │
          │   gRPC/HTTP           │
          │                       │
     ┌────┴───────────────────────┴────┐
     │        Host Machine             │
     │  localhost:9090 (Secret Mgr)    │
     │  localhost:8081 (Secret Mgr)    │
     │  localhost:9091 (KMS)           │
     │  localhost:8082 (KMS)           │
     │  localhost:8080 (IAM)           │
     └─────────────────────────────────┘
```

### Service Discovery

- **Within Docker network**: Services use container names (`iam:8080`, `secret-mgr:9090`)
- **From host**: Services use `localhost` with mapped ports
- **Health checks**: Docker uses `curl http://localhost:8080/health`

### Port Allocation Strategy

```
IAM Emulator:
  8080  - gRPC (standard)

Secret Manager:
  9090  - gRPC (data plane standard)
  8081  - HTTP (avoid conflict with IAM 8080)

KMS:
  9091  - gRPC (avoid conflict with Secret Manager)
  8082  - HTTP (avoid conflict with others)
```

---

## Data Flow

### Secret Creation with Encryption

**Scenario:** Create secret with KMS-encrypted value

```
Client
  │
  ├─ 1. Encrypt plaintext with KMS
  │    POST /v1/projects/test/locations/global/keyRings/app/cryptoKeys/data:encrypt
  │    X-Emulator-Principal: user:alice@example.com
  │    Body: {"plaintext": "c2VjcmV0"}
  │
  ▼
KMS Emulator
  │
  ├─ 2. Check permission: cloudkms.cryptoKeys.encrypt
  │    → TestIamPermissions(alice, projects/test/.../data, encrypt)
  │
  ▼
IAM Emulator
  │
  ├─ 3. Evaluate policy
  │    ✓ alice in developers group
  │    ✓ developers have encrypt permission
  │    → Return: ["cloudkms.cryptoKeys.encrypt"]
  │
  ▼
KMS Emulator
  │
  ├─ 4. Execute encryption
  │    → Return: {"ciphertext": "ZW5jcnlwdGVk"}
  │
  ▼
Client
  │
  ├─ 5. Store ciphertext in Secret Manager
  │    POST /v1/projects/test/secrets/db-password:addVersion
  │    X-Emulator-Principal: user:alice@example.com
  │    Body: {"payload": {"data": "ZW5jcnlwdGVk"}}
  │
  ▼
Secret Manager Emulator
  │
  ├─ 6. Check permission: secretmanager.versions.add
  │    → TestIamPermissions(alice, projects/test/secrets/db-password, versions.add)
  │
  ▼
IAM Emulator
  │
  ├─ 7. Evaluate policy
  │    ✓ alice has secretmanager.versions.add
  │    → Return: ["secretmanager.versions.add"]
  │
  ▼
Secret Manager Emulator
  │
  ├─ 8. Store version
  │    → Return: {version: "1"}
  │
  ▼
Client
```

**Permission checks:** 2 total (1 KMS encrypt, 1 Secret Manager add version)

---

## Design Decisions

### 1. Stateless Data Plane

**Decision:** Data plane emulators delegate all authorization to IAM emulator

**Rationale:**
- Single source of truth for policy
- No policy synchronization needed
- Easy to update policy without restarting data plane
- Matches real GCP architecture

**Trade-off:** Extra network hop for permission checks (acceptable for emulator)

### 2. Opt-In IAM Integration

**Decision:** IAM_MODE defaults to `off` (legacy behavior)

**Rationale:**
- Non-breaking for existing users
- Gradual migration path
- Clear opt-in signals intent to use IAM
- Flexibility for different environments

**Trade-off:** Users must explicitly enable IAM (acceptable, documented)

### 3. Metadata-Based Identity

**Decision:** Principal propagated via gRPC metadata, not request body

**Rationale:**
- Matches real GCP (identity in auth layer, not request)
- Enables SDK compatibility (no request modification)
- Separation of concerns (control vs data plane)
- Transparent forwarding

**Trade-off:** Extra header to remember (mitigated by library)

### 4. Fail-Open vs Fail-Closed Modes

**Decision:** Two modes (`permissive` and `strict`) for different use cases

**Rationale:**
- Development needs fail-open (don't block on IAM issues)
- CI needs fail-closed (catch permission bugs)
- Explicit choice forces consideration

**Trade-off:** More complexity (acceptable, well-documented)

### 5. In-Memory Storage

**Decision:** Emulators use in-memory storage, no persistence

**Rationale:**
- Simpler implementation
- Fast startup/teardown
- Hermetic testing (clean state per run)
- Emulator scope (not production replacement)

**Trade-off:** Data lost on restart (acceptable for testing)

---

## Extension Points

### Adding New Emulators

To add a new emulator to the control plane:

1. **Implement integration contract** (see [INTEGRATION_CONTRACT.md](INTEGRATION_CONTRACT.md))
2. **Add to docker-compose.yml**:
   ```yaml
   new-service:
     image: ghcr.io/blackwell-systems/gcp-new-service-emulator:latest
     ports:
       - "9092:9090"
     environment:
       - IAM_MODE=permissive
       - IAM_HOST=iam:8080
     depends_on:
       iam:
         condition: service_healthy
   ```
3. **Add permissions to policy packs** (`packs/new-service.yaml`)
4. **Update documentation** (README, tutorial)

### Custom Policy Sources

**Current:** `policy.yaml` file mount

**Future extension points:**
- Git repository sync
- Secret Manager policy storage
- Dynamic policy reload
- Policy inheritance/layering

### Advanced CEL Conditions

**Current:** Basic CEL expressions on `resource.name`

**Future capabilities:**
- Time-based conditions (`request.time`)
- Tag-based conditions (`resource.tags`)
- Custom attributes
- External data sources

### Observability Integration

**Current:** Docker logs

**Future extension points:**
- OpenTelemetry traces
- Prometheus metrics
- Audit log export
- Permission check visualization

---

## Performance Characteristics

### Latency Budget

```
Operation without IAM:  ~1ms   (in-memory)
Permission check:       ~5ms   (network + policy eval)
Total with IAM:         ~6ms   (acceptable for testing)
```

### Scalability

**Designed for:**
- Single developer workstation
- CI/CD test runners
- Integration test suites

**NOT designed for:**
- Production workloads
- High throughput (1000+ req/s)
- Large datasets (>10k resources)

### Resource Usage

```
IAM Emulator:         ~50MB RAM
Secret Manager:       ~30MB RAM
KMS:                  ~30MB RAM
Total:                ~110MB RAM (acceptable for Docker)
```

---

## Security Model

### Threat Model

**In scope:**
- Authorization logic correctness
- Permission denial enforcement
- Condition evaluation accuracy

**Out of scope:**
- Authentication (no real auth in emulator)
- Encryption strength (KMS uses weak crypto)
- Data persistence security (in-memory only)
- Network security (assumes trusted network)

### Design for Testing, Not Production

The control plane is explicitly **NOT production-ready**:
- No TLS/encryption
- No authentication
- No audit logging
- No rate limiting
- No data persistence

**Use case:** Testing IAM behavior in development/CI environments

---

## Related Documentation

- [Integration Contract](INTEGRATION_CONTRACT.md) - Technical contract for emulators
- [End-to-End Tutorial](END_TO_END_TUTORIAL.md) - Complete usage walkthrough
- [Troubleshooting](TROUBLESHOOTING.md) - Common issues and solutions
