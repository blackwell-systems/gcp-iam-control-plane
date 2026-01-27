# GCP Emulator Control Plane

[![Blackwell Systems](https://raw.githubusercontent.com/blackwell-systems/blackwell-docs-theme/main/badge-trademark.svg)](https://github.com/blackwell-systems)
[![Go Reference](https://pkg.go.dev/badge/github.com/blackwell-systems/gcp-emulator-control-plane.svg)](https://pkg.go.dev/github.com/blackwell-systems/gcp-emulator-control-plane)
[![Go Version](https://img.shields.io/badge/go-1.24+-blue.svg)](https://go.dev/)
[![Test Status](https://github.com/blackwell-systems/gcp-emulator-control-plane/workflows/Test/badge.svg)](https://github.com/blackwell-systems/gcp-emulator-control-plane/actions)
[![Version](https://img.shields.io/github/v/release/blackwell-systems/gcp-emulator-control-plane)](https://github.com/blackwell-systems/gcp-emulator-control-plane/releases)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

> **If you're testing GCP emulators without IAM, you're not testing production behavior.**  
> This repo adds the missing control plane.

> One policy file. One principal injection method. Consistent authorization across all emulators.

**GCP Emulator Control Plane** is the orchestration repo for the Blackwell Systems local GCP emulator ecosystem.

## Ecosystem Components

This control plane orchestrates the complete Blackwell Systems GCP emulator ecosystem:

| Component | Role | Repository | Status |
|-----------|------|------------|--------|
| [gcp-iam-emulator](https://github.com/blackwell-systems/gcp-iam-emulator) | Authorization engine (control plane) | [`blackwell-systems/gcp-iam-emulator`](https://github.com/blackwell-systems/gcp-iam-emulator) | ✓ Stable |
| [gcp-secret-manager-emulator](https://github.com/blackwell-systems/gcp-secret-manager-emulator) | Secret Manager API (data plane) | [`blackwell-systems/gcp-secret-manager-emulator`](https://github.com/blackwell-systems/gcp-secret-manager-emulator) | ✓ Stable (v1.2.1+) |
| [gcp-kms-emulator](https://github.com/blackwell-systems/gcp-kms-emulator) | KMS API (data plane) | [`blackwell-systems/gcp-kms-emulator`](https://github.com/blackwell-systems/gcp-kms-emulator) | ✓ Stable (v0.2.1+) |

**All components can run standalone or orchestrated together.**

Each emulator follows the [Integration Contract](docs/INTEGRATION_CONTRACT.md):
- Resource naming conventions
- Permission mappings
- Principal propagation (gRPC + HTTP)
- IAM mode configuration (off, permissive, strict)

Future emulators (Cloud Storage, Pub/Sub, etc.) will follow the same contract

This repo provides the **control plane glue**:
- **`gcp-emulator` CLI** - Single command to manage the entire stack
- `docker compose` orchestration for direct usage
- Single `policy.yaml` that drives authorization everywhere
- Stable **integration contract** (resource naming + permissions + principal propagation)
- End-to-end examples and integration tests that mirror production IAM behavior

---

## Why This Exists

Most "emulators" are **data-plane only**: they implement CRUD operations but skip auth.

In real GCP, **IAM is the control plane**:
- every request is authorized
- conditions restrict access (resource name, time, etc.)
- policies are inherited down resource hierarchies

If your tests don't exercise authorization, you miss an entire class of production bugs:
- incorrect roles
- missing permissions
- wrong principal identity
- policies that pass in dev but fail in prod

**This repo makes IAM enforcement testable and deterministic, locally and in CI.**

---

## What You Get

### One policy file (offline, deterministic)
Define your authorization universe once in `policy.yaml`.

### One identity channel end-to-end
Inject a principal consistently:
- gRPC: `x-emulator-principal`
- HTTP: `X-Emulator-Principal`

That identity is propagated from emulator → IAM emulator without rewriting your app code.

### Cross-service authorization
Secret Manager and KMS enforce the same policy engine, the same way.

### CI-friendly and hermetic
No network calls, no cloud credentials required.

### Unified CLI
Single command to manage the entire stack - no docker-compose knowledge required.

---

## Quickstart

### Option 1: Using the CLI (Recommended)

**Install:**
```bash
go install github.com/blackwell-systems/gcp-emulator-control-plane/cmd/gcp-emulator@latest
```

**Start the stack:**
```bash
gcp-emulator start
```

**Check status:**
```bash
gcp-emulator status
```

You now have:
- IAM Emulator: `localhost:8080` (gRPC)
- Secret Manager Emulator: `localhost:9090` (gRPC), `localhost:8081` (HTTP)
- KMS Emulator: `localhost:9091` (gRPC), `localhost:8082` (HTTP)

### Option 2: Using Docker Compose Directly

**Prerequisites:**
- Docker + Docker Compose

**Start the stack:**
```bash
docker compose up
```

### 3) Configure policy

Edit `policy.yaml`:

```yaml
roles:
  roles/custom.ciRunner:
    permissions:
      - secretmanager.secrets.get
      - secretmanager.versions.access
      - cloudkms.cryptoKeys.encrypt

groups:
  developers:
    members:
      - user:alice@example.com
      - user:bob@example.com

projects:
  test-project:
    bindings:
      - role: roles/owner
        members:
          - group:developers

      - role: roles/custom.ciRunner
        members:
          - serviceAccount:ci@test-project.iam.gserviceaccount.com
        condition:
          expression: 'resource.name.startsWith("projects/test-project/secrets/prod-")'
          title: "CI limited to production secrets"
```

### 4) Test principal injection

**Using the CLI:**
```bash
# Check status
gcp-emulator status

# View logs
gcp-emulator logs --follow
```

**HTTP example (Secret Manager):**

```bash
curl -X POST http://localhost:8081/v1/projects/test-project/secrets \
  -H "X-Emulator-Principal: user:alice@example.com" \
  -H "Content-Type: application/json" \
  -d '{"secretId":"db-password","payload":{"data":"c2VjcmV0"}}'
```

**Check IAM logs:**

```bash
# With CLI
gcp-emulator logs iam

# Or with docker-compose
docker compose logs iam
```

---

## Why not just mock the SDK?

Because mocks don't test:
- inheritance resolution
- conditional bindings
- cross-service consistency
- real permission names / drift

This stack tests the actual control plane behavior.

---

## Stack Overview

### Control Plane

- **IAM Emulator**: policy evaluation engine
- **policy.yaml**: the source of truth for authorization behavior
- **principal propagation**: consistent identity channel

### Data Plane

- **Secret Manager Emulator**: enforces `secretmanager.*` permissions
- **KMS Emulator**: enforces `cloudkms.*` permissions

---

## Integration Contract (Stable)

This repo defines the contract new emulators must implement to join the mesh.

### 1) Canonical resource naming

**Secret Manager**

```
projects/{project}/secrets/{secret}
projects/{project}/secrets/{secret}/versions/{version}
```

**KMS**

```
projects/{project}/locations/{location}/keyRings/{keyring}
projects/{project}/locations/{location}/keyRings/{keyring}/cryptoKeys/{key}
projects/{project}/locations/{location}/keyRings/{keyring}/cryptoKeys/{key}/cryptoKeyVersions/{version}
```

### 2) Operation → permission mapping

Each emulator maps API operations to real GCP permissions.

Examples:
- `AccessSecretVersion` → `secretmanager.versions.access`
- `Encrypt` → `cloudkms.cryptoKeys.encrypt`

### 3) Principal injection (inbound)

- gRPC: `x-emulator-principal`
- HTTP: `X-Emulator-Principal`

### 4) Principal propagation (outbound)

Emulators call IAM emulator using `TestIamPermissions`, and propagate identity via metadata (not request fields).

---

## Compatibility and Non-Breaking Behavior

IAM enforcement is **opt-in** per emulator.

Default behavior remains the same as classic emulators:
- IAM disabled → all requests succeed (legacy behavior)

When enabled:
- permissive or strict mode controls failure behavior (fail-open vs fail-closed)

**Environment variables (standardized):**

| Variable     | Purpose           | Default              |
| ------------ | ----------------- | -------------------- |
| `IAM_MODE`   | off/permissive/strict | `off`            |
| `IAM_HOST`   | IAM endpoint      | `iam:8080` (compose) |

---

## CLI Commands

The `gcp-emulator` CLI provides a unified interface:

**Stack management:**
```bash
gcp-emulator start [--mode=permissive|strict|off]
gcp-emulator stop
gcp-emulator restart [service]
gcp-emulator status
gcp-emulator logs [service] [--follow]
```

**Policy management:**
```bash
gcp-emulator policy validate [file]
gcp-emulator policy init [--template=basic|advanced|ci]
```

**Configuration:**
```bash
gcp-emulator config get
gcp-emulator config set <key> <value>
gcp-emulator config reset
```

**For complete CLI documentation, see [CLI_DESIGN.md](docs/CLI_DESIGN.md)**

---

## Repo Layout

```
.
├─ cmd/gcp-emulator/           # CLI entry point
├─ internal/
│   ├─ cli/                    # CLI commands
│   ├─ config/                 # Configuration (Viper)
│   ├─ docker/                 # Docker compose wrapper
│   └─ policy/                 # Policy parsing/validation
├─ docker-compose.yml
├─ policy.yaml
├─ packs/
│   ├─ secretmanager.yaml
│   ├─ kms.yaml
│   └─ ci.yaml
├─ examples/
│   ├─ go/
│   └─ curl/
├─ docs/
│   ├─ CLI_DESIGN.md
│   ├─ CLI_VIPER_PATTERN.md
│   ├─ END_TO_END_TUTORIAL.md
│   ├─ ARCHITECTURE.md
│   ├─ MIGRATION.md
│   ├─ TROUBLESHOOTING.md
│   └─ INTEGRATION_CONTRACT.md
└─ README.md
```

---

## Policy Packs

The `packs/` directory contains ready-to-copy role definitions for common services:
- Secret Manager roles
- KMS roles
- CI patterns

Start simple: copy/paste into your `policy.yaml`.

(Directory merge/import can be added later if demand exists.)

---

## CI Usage

### Using the CLI in CI

```yaml
- name: Install gcp-emulator CLI
  run: go install github.com/blackwell-systems/gcp-emulator-control-plane/cmd/gcp-emulator@latest

- name: Start emulator stack
  run: gcp-emulator start --mode=strict

- name: Run tests
  run: go test ./...

- name: Check IAM logs for denials
  if: failure()
  run: gcp-emulator logs iam | grep DENY

- name: Stop emulators
  run: gcp-emulator stop
```

### Using Docker Compose in CI

```yaml
- name: Start emulators
  run: docker compose up -d

- name: Run tests
  run: go test ./...

- name: Stop emulators
  run: docker compose down
```

---

## Roadmap

- Add more policy examples and end-to-end tutorials
- Publish "known good" policy templates for common stacks
- Add additional emulators following the contract (Pub/Sub, Storage, etc.)
- Provide an integration test harness repo for emulator authors

---

## Disclaimer

This project is not affiliated with, endorsed by, or sponsored by Google LLC.
"Google Cloud", "GCP", "IAM", and related trademarks are property of Google LLC.

---

## Maintained By

Maintained by **Dayna Blackwell** — founder of Blackwell Systems.

- GitHub: [https://github.com/blackwell-systems](https://github.com/blackwell-systems)
- Blog: [https://blog.blackwell-systems.com](https://blog.blackwell-systems.com)
- LinkedIn: [https://linkedin.com/in/dayna-blackwell](https://linkedin.com/in/dayna-blackwell)
