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

Unlike mocks (which allow everything) or observers (which record after the fact), the **Blackwell IAM Control Plane** actively denies unauthorized requests before they reach emulators.

| Approach | Example | When | Behavior |
|----------|---------|------|----------|
| Mock | Standard emulators | Never | Always allows |
| Observer | Post-execution analysis | After | Records what you used |
| **Control Plane** | **Blackwell IAM** | **Before** | **Denies unauthorized** |

Pre-flight enforcement catches permission bugs in development and CI, not production.

---

## The Missing Hermetic Seal

Before Blackwell, **"GCP Hermetic Testing" was essentially impossible.**

While Google has long provided emulators for individual data plane services (Pub/Sub, Spanner, BigTable), they intentionally left a massive hole where the **Identity Layer** should be.

**The two bad options you had:**

1. **Fake Auth** - Official emulators ignore permissions (tests pass locally, fail in production)
2. **Staging Leak** - Call real GCP IAM API (hermetic seal broken, tests become flaky with 1-60s propagation delays)

**Blackwell closes the hermetic seal:**

This control plane provides what was missing:
- **Deterministic IAM** - Strongly consistent (0ms delay vs 1-60s in real GCP)
- **Offline Authorization** - Services check permissions locally (no network required)
- **Zero-Credential Loop** - Simulate identities in-memory (no service account keys)

**The result:** Tests fail exactly like production, run completely offline, execute deterministically.

**The Security Paradox:**
> "A test that cannot fail due to a permission error is a test that has not fully validated the code's production readiness."

This is not an incremental improvement. This is the **completion of a system that was deliberately left incomplete**.

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

## Authorization Tracing

Structured logging of IAM authorization decisions across the entire emulator stack.

### Enable Tracing

```bash
# Start stack with tracing enabled
IAM_TRACE_OUTPUT=./authz-trace.jsonl gcp-emulator start --mode=strict

# Or export environment variable
export IAM_TRACE_OUTPUT=./authz-trace.jsonl
gcp-emulator start --mode=strict
```

### What Gets Traced

Every authorization check across all services emits structured events:
- **IAM Emulator** (control plane) - Policy engine decisions (authoritative)
- **Secret Manager** - Secret access checks (via enforcement proxy)
- **KMS** - Key operation checks (via enforcement proxy)

Each service emits JSONL events with principal, resource, permission, outcome, and timing.

### Use Cases

**Debug Cross-Service Authorization:**
```bash
# See all denied requests across the stack
cat authz-trace.jsonl | jq 'select(.decision.outcome=="DENY")'

# Filter by service
cat authz-trace.jsonl | jq 'select(.environment.component=="gcp-iam-emulator")'
```

**Audit CI Access Patterns:**
```bash
# List all resources accessed during test run
cat authz-trace.jsonl | \
  jq -r 'select(.decision.outcome=="ALLOW") | .target.resource' | sort -u

# See which principals were tested
cat authz-trace.jsonl | jq -r '.actor.principal' | sort -u
```

**Validate Policy Changes:**
```bash
# Before policy change
IAM_TRACE_OUTPUT=./before.jsonl gcp-emulator start --mode=strict
go test ./...
gcp-emulator stop

# After policy change
IAM_TRACE_OUTPUT=./after.jsonl gcp-emulator start --mode=strict
go test ./...

# Compare outcomes (detect regressions)
diff <(jq -r '.decision.outcome' before.jsonl | sort) \
     <(jq -r '.decision.outcome' after.jsonl | sort)
```

**CI/CD Compliance:**
```bash
# GitHub Actions - archive traces as artifacts
- name: Run tests with tracing
  env:
    IAM_TRACE_OUTPUT: ${{ runner.temp }}/authz-trace.jsonl
  run: |
    gcp-emulator start --mode=strict
    go test ./...
    
- name: Upload authorization audit trail
  uses: actions/upload-artifact@v3
  with:
    name: authorization-traces
    path: ${{ runner.temp }}/authz-trace.jsonl
```

### Trace Schema

Events follow schema v1.0:
- JSONL format (one event per line)
- Schema-versioned for compatibility
- Includes actor, resource, permission, decision, timing, component

**Example event:**
```json
{"schema_version":"1.0","event_type":"authz_check","timestamp":"2026-01-28T10:15:23.483Z","actor":{"principal":"user:alice@example.com"},"target":{"resource":"projects/test/secrets/db-password"},"action":{"permission":"secretmanager.secrets.get"},"decision":{"outcome":"ALLOW","reason":"binding_match","evaluated_by":"gcp-iam-emulator","latency_ms":3}}
```

See component READMEs for detailed schema documentation:
- [IAM Emulator Tracing](https://github.com/blackwell-systems/gcp-iam-emulator#authorization-tracing)
- [Enforcement Proxy Tracing](https://github.com/blackwell-systems/gcp-emulator-auth#authorization-tracing)

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

This architecture matches real GCP. Secrets/keys live in the data plane. Authorization logic lives in the control plane. Testing this separation catches production bugs.

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
