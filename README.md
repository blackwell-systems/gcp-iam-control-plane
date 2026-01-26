# GCP Emulator Control Plane

> **If you're testing GCP emulators without IAM, you're not testing production behavior.**  
> This repo adds the missing control plane.

> One policy file. One principal injection method. Consistent authorization across all emulators.

**GCP Emulator Control Plane** is the orchestration repo for the Blackwell Systems local GCP emulator ecosystem.

It composes:
- **gcp-iam-emulator** (authorization engine)
- **gcp-secret-manager-emulator** (data plane)
- **gcp-kms-emulator** (data plane)
- (future) additional emulators that follow the same contract

This repo provides the **control plane glue**:
- `docker compose up` to run a coherent stack
- a single `policy.yaml` that drives authorization everywhere
- a stable **integration contract** (resource naming + permissions + principal propagation)
- end-to-end examples and integration tests that mirror production IAM behavior

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

---

## Quickstart

### 1) Prerequisites
- Docker + Docker Compose
- (optional) Go toolchain if you want to build locally

### 2) Start the stack

```bash
docker compose up
```

You now have:
- IAM Emulator: `localhost:8080` (gRPC)
- Secret Manager Emulator: `localhost:9090` (gRPC), `localhost:8081` (HTTP)
- KMS Emulator: `localhost:9091` (gRPC), `localhost:8082` (HTTP)

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

**HTTP example (Secret Manager):**

```bash
curl -X POST http://localhost:8081/v1/projects/test-project/secrets \
  -H "X-Emulator-Principal: user:alice@example.com" \
  -H "Content-Type: application/json" \
  -d '{"secretId":"db-password","payload":{"data":"c2VjcmV0"}}'
```

Check IAM logs:

```bash
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

## Repo Layout

```
.
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
│   ├─ INTEGRATION_CONTRACT.md
│   ├─ MIGRATION.md
│   └─ TROUBLESHOOTING.md
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

### GitHub Actions (example)

```yaml
services:
  control-plane:
    image: ghcr.io/blackwell-systems/gcp-emulator-control-plane:latest
```

Or run directly with compose:

```yaml
- name: Start emulators
  run: docker compose up -d

- name: Run tests
  run: go test ./...
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
- LinkedIn: [https://linkedin.com/in/daynablackwell](https://linkedin.com/in/daynablackwell)
