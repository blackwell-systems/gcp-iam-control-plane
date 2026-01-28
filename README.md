# GCP IAM Control Plane

[![Blackwell Systems](https://raw.githubusercontent.com/blackwell-systems/blackwell-docs-theme/main/badge-trademark.svg)](https://github.com/blackwell-systems)
[![Go Reference](https://pkg.go.dev/badge/github.com/blackwell-systems/gcp-iam-control-plane.svg)](https://pkg.go.dev/github.com/blackwell-systems/gcp-iam-control-plane)
[![Go Version](https://img.shields.io/badge/go-1.24+-blue.svg)](https://go.dev/)
[![Test Status](https://github.com/blackwell-systems/gcp-iam-control-plane/workflows/Test/badge.svg)](https://github.com/blackwell-systems/gcp-iam-control-plane/actions)
[![Version](https://img.shields.io/github/v/release/blackwell-systems/gcp-iam-control-plane)](https://github.com/blackwell-systems/gcp-iam-control-plane/releases)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

> **Enforce real GCP IAM policies in local development and CI** — Make your emulators fail exactly like production would.

Orchestrates the **Local IAM Control Plane** — a CLI (`gcp-emulator`) that manages GCP service emulators with pre-flight IAM enforcement. Start/stop services, manage policies, and test authorization without cloud credentials or docker-compose knowledge.

---

## What This Is

Unlike mocks (which allow everything) or observers like iamlive (which record after the fact), the **Blackwell IAM Control Plane** actively denies unauthorized requests before they reach emulators.

| Approach | Example | When | Behavior |
|----------|---------|------|----------|
| Mock | Standard emulators | Never | Always allows |
| Observer | iamlive (AWS) | After | Records what you used |
| **Control Plane** | **Blackwell IAM** | **Before** | **Denies unauthorized** |

**Key insight:** Pre-flight enforcement catches permission bugs in development and CI, not production.

---

## Architecture

```
┌─────────────────────────────────────────┐
│  Your Application Code                  │
│  (GCP client libraries)                 │
└────────────────┬────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────┐
│  DATA PLANES (Started by this CLI)     │
│  • Secret Manager Emulator              │
│  • KMS Emulator                         │
│  • (Future: Tasks, Pub/Sub, Storage)    │
│                                         │
│  Each checks IAM before data access     │
└────────────────┬────────────────────────┘
                 │
                 │ CheckPermission(principal, resource, permission)
                 ▼
┌─────────────────────────────────────────┐
│  CONTROL PLANE (Policy Engine)          │
│  IAM Emulator                           │
│                                         │
│  - Role bindings                        │
│  - Group memberships                    │
│  - Policy inheritance                   │
│  - Deterministic evaluation             │
└─────────────────────────────────────────┘
```


This CLI orchestrates the IAM control plane and all data plane services.

---

## Why This Exists

Most GCP emulators skip authorization entirely. Without IAM testing, you discover permission bugs **after deployment**:
- Incorrect role assignments
- Missing permissions  
- Wrong principal identity
- Policies that work in dev but fail in prod

**This control plane makes IAM enforcement testable and deterministic**, locally and in CI.

---

## What You Get

- **Unified CLI** - Single command to start/stop/restart the entire stack, no docker-compose knowledge required
- **Production policy testing** - Export real GCP policies and test locally (`gcloud get-iam-policy` → test)
- **One policy file** - Define authorization once in `policy.yaml` or `policy.json`, enforced consistently across all emulators
- **Principal injection** - Consistent identity channel (gRPC `x-emulator-principal`, HTTP `X-Emulator-Principal`)
- **Hermetic testing** - No network calls, no cloud credentials, deterministic CI pipelines
- **Multiple IAM modes** - Off/permissive/strict for different testing scenarios

---

## Quickstart

### Install

```bash
go install github.com/blackwell-systems/gcp-iam-control-plane/cmd/gcp-emulator@latest
```

### Start the Stack

```bash
gcp-emulator start
```

This starts three services:
- **IAM Emulator** (control plane): `localhost:8080` (gRPC)
- **Secret Manager Emulator** (data plane): `localhost:9090` (gRPC), `localhost:8081` (HTTP)
- **KMS Emulator** (data plane): `localhost:9091` (gRPC), `localhost:8082` (HTTP)

### Configure Policy

Edit `policy.yaml` (YAML or JSON supported):

```yaml
roles:
  roles/custom.developer:
    permissions:
      - secretmanager.secrets.get
      - secretmanager.versions.access

groups:
  developers:
    members:
      - user:alice@example.com

projects:
  test-project:
    bindings:
      - role: roles/custom.developer
        members:
          - group:developers
```

See [Policy Reference](docs/POLICY_REFERENCE.md) for complete policy syntax.

### Test with Principal Injection

```bash
# Secret Manager HTTP API
curl -X POST http://localhost:8081/v1/projects/test-project/secrets \
  -H "X-Emulator-Principal: user:alice@example.com" \
  -H "Content-Type: application/json" \
  -d '{"secretId":"db-password","payload":{"data":"c2VjcmV0"}}'

# Check authorization logs
gcp-emulator logs iam
```

---

## Testing with Production Policies

Stop hand-writing test policies that drift from production.

Export your real GCP IAM policy and test against it:

```bash
# Export production policy
gcloud projects get-iam-policy my-prod-project --format=json > prod-policy.json

# Test locally with production permissions
gcp-emulator start --policy-file=prod-policy.json --mode=strict
go test ./...

# Catch permission issues before deploying
gcp-emulator logs iam | grep DENY
```

**Why this matters:**
- Test with exact production policy
- Catch permission issues in CI, not production
- No policy drift between environments
- Works with Terraform, CDK, or Console exports

Both YAML and JSON formats supported. JSON matches GCP's native IAM policy structure.

---

## Ecosystem

This control plane orchestrates multiple emulators:

| Component | Role | Status |
|-----------|------|--------|
| [gcp-iam-emulator](https://github.com/blackwell-systems/gcp-iam-emulator) | Authorization engine (control plane) | ✓ Stable |
| [gcp-secret-manager-emulator](https://github.com/blackwell-systems/gcp-secret-manager-emulator) | Secret Manager API (data plane) | ✓ Stable |
| [gcp-kms-emulator](https://github.com/blackwell-systems/gcp-kms-emulator) | KMS API (data plane) | ✓ Stable |

All components can run standalone or orchestrated together. New emulators follow the [Integration Contract](docs/INTEGRATION_CONTRACT.md).

---

## CLI Commands

```bash
# Stack management
gcp-emulator start [--mode=permissive|strict|off]
gcp-emulator stop
gcp-emulator status
gcp-emulator logs [service] [--follow]

# Policy management
gcp-emulator policy validate [file]
gcp-emulator policy init [--template=basic|advanced|ci] [--output=policy.yaml]

# Configuration
gcp-emulator config get
gcp-emulator config set <key> <value>
```

See [CLI Design](docs/CLI_DESIGN.md) for complete command reference.

---

## CI Integration

### GitHub Actions

```yaml
- name: Install CLI
  run: go install github.com/blackwell-systems/gcp-iam-control-plane/cmd/gcp-emulator@latest

- name: Start emulators (strict mode)
  run: gcp-emulator start --mode=strict

- name: Run tests
  run: go test ./...

- name: Check IAM logs for denials
  if: failure()
  run: gcp-emulator logs iam | grep DENY

- name: Stop emulators
  if: always()
  run: gcp-emulator stop
```

**IAM Modes:**
- `off` - No IAM enforcement (fast iteration)
- `permissive` - IAM enabled, fail-open on errors (development)
- `strict` - IAM enabled, fail-closed (production parity, recommended for CI)

See [CI Integration](docs/CI_INTEGRATION.md) for GitLab, CircleCI, Jenkins examples.

---

## Control Plane vs Data Plane

**Control Plane (IAM Emulator):**
- Evaluates authorization policy from `policy.yaml`
- Expands roles, resolves groups, evaluates conditions
- Returns allow/deny decisions
- Stateless - doesn't know about secrets or keys

**Data Plane (Secret Manager, KMS):**
- Implements CRUD operations
- Checks permissions by calling IAM emulator
- Stores resources in-memory
- Enforces decisions from control plane

**Key insight:** This architecture matches real GCP. Secrets/keys live in the data plane. Authorization logic lives in the control plane. Testing this separation catches production bugs.

See [Architecture](docs/ARCHITECTURE.md) for detailed system design.

---

## Why Not Mock the SDK?

Mocks don't test:
- **Policy inheritance** - Project-level bindings affecting resources
- **Conditional access** - CEL expressions restricting by resource name
- **Cross-service consistency** - Same policy engine for all services
- **Permission drift** - Real GCP permission names

This stack tests **actual control plane behavior**.

---

## Docker Compose (Manual)

If you prefer direct orchestration:

```bash
docker compose up -d   # Start stack
docker compose logs    # View logs
docker compose down    # Stop stack
```

The CLI wraps these commands with policy validation, status checks, and unified logging.

---

## Policy Packs

Ready-made role definitions in `packs/`:

```bash
# Copy Secret Manager roles
cat packs/secretmanager.yaml >> policy.yaml

# Copy KMS roles
cat packs/kms.yaml >> policy.yaml

# Copy CI patterns
cat packs/ci.yaml >> policy.yaml
```

See [Policy Reference](docs/POLICY_REFERENCE.md) for available packs and examples.

---

## Documentation

### Core Docs
- **[Policy Reference](docs/POLICY_REFERENCE.md)** - Complete policy syntax, conditions, permissions, examples
- **[CI Integration](docs/CI_INTEGRATION.md)** - GitHub Actions, GitLab, CircleCI, Jenkins examples
- **[Architecture](docs/ARCHITECTURE.md)** - Control plane design, request flow, authorization model

### Additional Resources
- **[Integration Contract](docs/INTEGRATION_CONTRACT.md)** - Contract for building new emulators
- **[CLI Design](docs/CLI_DESIGN.md)** - CLI implementation and Viper pattern
- **[End-to-End Tutorial](docs/END_TO_END_TUTORIAL.md)** - Complete usage walkthrough
- **[Troubleshooting](docs/TROUBLESHOOTING.md)** - Common issues and solutions
- **[Migration Guide](docs/MIGRATION.md)** - Migrating from standalone emulators

---

## Example: Conditional Access

Restrict CI to production secrets only:

```yaml
roles:
  roles/custom.ciRunner:
    permissions:
      - secretmanager.secrets.get
      - secretmanager.versions.access

projects:
  test-project:
    bindings:
      - role: roles/custom.ciRunner
        members:
          - serviceAccount:ci@test-project.iam.gserviceaccount.com
        condition:
          expression: 'resource.name.startsWith("projects/test-project/secrets/prod-")'
          title: "CI limited to production secrets"
```

Now CI can only access secrets starting with `prod-`:
- ✓ `projects/test-project/secrets/prod-db-password` - Allowed
- ✗ `projects/test-project/secrets/dev-api-key` - Denied (403)

See [Policy Reference](docs/POLICY_REFERENCE.md) for CEL condition syntax.

---

## Compatibility

IAM enforcement is **opt-in**. Default behavior matches standalone emulators:
- `IAM_MODE=off` - All requests succeed (legacy behavior)
- `IAM_MODE=permissive` - IAM enabled, fail-open on errors
- `IAM_MODE=strict` - IAM enabled, fail-closed

Non-breaking by design.

---

## Disclaimer

This project is not affiliated with, endorsed by, or sponsored by Google LLC. "Google Cloud", "GCP", "IAM", and related trademarks are property of Google LLC.

---

## Maintained By

**Dayna Blackwell** — Founder, Blackwell Systems

- GitHub: [https://github.com/blackwell-systems](https://github.com/blackwell-systems)
- Blog: [https://blog.blackwell-systems.com](https://blog.blackwell-systems.com)
- LinkedIn: [https://linkedin.com/in/dayna-blackwell](https://linkedin.com/in/dayna-blackwell)
