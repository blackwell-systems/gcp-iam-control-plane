# End-to-End Tutorial: Building a GCP-Like Local Environment with IAM

This tutorial walks through setting up the complete GCP emulator control plane with IAM-enforced authorization, from zero to running a multi-service application with realistic permission checks.

**What you'll build:**
- IAM emulator with custom roles and policies
- Secret Manager emulator with permission-based access control
- KMS emulator with encryption key authorization
- A sample application that uses both services with proper identity propagation

**Time estimate:** 30 minutes

---

## Prerequisites

- Docker and Docker Compose installed
- `curl` for testing REST endpoints
- (Optional) Go 1.24+ for SDK examples
- Basic understanding of GCP IAM concepts (principals, roles, permissions)

---

## Part 1: Understanding the Architecture

The GCP emulator control plane consists of three layers:

### Control Plane
- **IAM Emulator**: Policy evaluation engine that enforces permissions
- **policy.yaml**: Single source of truth for all authorization decisions

### Data Plane
- **Secret Manager Emulator**: Stores secrets, enforces `secretmanager.*` permissions
- **KMS Emulator**: Manages encryption keys, enforces `cloudkms.*` permissions

### Identity Propagation
- Principal injected via headers: `x-emulator-principal` (gRPC) or `X-Emulator-Principal` (HTTP)
- Data plane emulators forward identity to IAM emulator for permission checks
- IAM emulator evaluates policy and returns allow/deny

```
┌─────────────┐
│   Client    │
│ (curl/SDK)  │
└──────┬──────┘
       │ X-Emulator-Principal: user:alice@example.com
       ↓
┌─────────────────────┐
│  Secret Manager     │
│  or KMS Emulator    │ ←─── Extracts principal from headers
└──────┬──────────────┘
       │ CheckPermission(principal, resource, permission)
       ↓
┌─────────────────────┐
│   IAM Emulator      │
│  (policy.yaml)      │ ←─── Evaluates policy, returns allow/deny
└─────────────────────┘
```

---

## Part 2: Setting Up the Stack

### Step 1: Clone the Control Plane Repository

```bash
git clone https://github.com/blackwell-systems/gcp-emulator-control-plane.git
cd gcp-emulator-control-plane
```

### Step 2: Review the Default Policy

Open `policy.yaml`:

```yaml
roles:
  roles/custom.developer:
    permissions:
      - secretmanager.secrets.create
      - secretmanager.secrets.get
      - secretmanager.versions.add
      - secretmanager.versions.access
      - cloudkms.cryptoKeys.encrypt
      - cloudkms.cryptoKeys.decrypt

  roles/custom.ciRunner:
    permissions:
      - secretmanager.secrets.get
      - secretmanager.versions.access

groups:
  developers:
    members:
      - user:alice@example.com
      - user:bob@example.com

projects:
  test-project:
    bindings:
      - role: roles/custom.developer
        members:
          - group:developers

      - role: roles/custom.ciRunner
        members:
          - serviceAccount:ci@test-project.iam.gserviceaccount.com
        condition:
          expression: 'resource.name.startsWith("projects/test-project/secrets/prod-")'
          title: "CI limited to production secrets only"
```

**Key concepts:**
- **Custom roles**: Define exactly which permissions you need
- **Groups**: Organize users for easier policy management
- **Bindings**: Assign roles to principals (users, service accounts, groups)
- **Conditions**: CEL expressions that restrict when permissions apply

### Step 3: Start the Stack

```bash
docker compose up -d
```

**Services started:**
- IAM Emulator: `localhost:8080`
- Secret Manager: `localhost:9090` (gRPC), `localhost:8081` (HTTP)
- KMS: `localhost:9091` (gRPC), `localhost:8082` (HTTP)

### Step 4: Verify Health

```bash
curl http://localhost:8080/health
curl http://localhost:8081/health
curl http://localhost:8082/health
```

All should return `{"status":"ok"}`.

---

## Part 3: Testing Secret Manager with IAM

### Scenario: Alice creates and accesses a secret

Alice is in the `developers` group, which has the `roles/custom.developer` role.

#### Create a secret

```bash
curl -X POST http://localhost:8081/v1/projects/test-project/secrets?secretId=db-password \
  -H "X-Emulator-Principal: user:alice@example.com" \
  -H "Content-Type: application/json" \
  -d '{
    "replication": {
      "automatic": {}
    },
    "labels": {
      "env": "dev",
      "team": "backend"
    }
  }'
```

**Expected:** Success (200 OK)

**Why:** Alice has `secretmanager.secrets.create` permission via `roles/custom.developer`

#### Add a secret version

```bash
curl -X POST http://localhost:8081/v1/projects/test-project/secrets/db-password:addVersion \
  -H "X-Emulator-Principal: user:alice@example.com" \
  -H "Content-Type: application/json" \
  -d '{
    "payload": {
      "data": "'$(echo -n "super-secret-password" | base64)'"
    }
  }'
```

**Expected:** Success (200 OK)

**Why:** Alice has `secretmanager.versions.add` permission

#### Access the secret

```bash
curl http://localhost:8081/v1/projects/test-project/secrets/db-password/versions/latest:access \
  -H "X-Emulator-Principal: user:alice@example.com"
```

**Expected:** Returns the payload with base64-encoded data

**Verify decoding:**
```bash
curl -s http://localhost:8081/v1/projects/test-project/secrets/db-password/versions/latest:access \
  -H "X-Emulator-Principal: user:alice@example.com" \
  | jq -r '.payload.data' | base64 -d
```

**Output:** `super-secret-password`

---

## Part 4: Testing Permission Denials

### Scenario: Unauthorized user tries to access secrets

Charlie is not in any group and has no permissions.

```bash
curl http://localhost:8081/v1/projects/test-project/secrets/db-password/versions/latest:access \
  -H "X-Emulator-Principal: user:charlie@example.com"
```

**Expected:** `403 Forbidden` or `Permission denied`

**Check IAM logs:**
```bash
docker compose logs iam | grep charlie
```

You'll see the denial logged with the exact permission check that failed.

---

## Part 5: Testing Conditional Permissions

### Scenario: CI service account with restricted access

The `ci@test-project.iam.gserviceaccount.com` principal has `roles/custom.ciRunner`, but only for secrets starting with `prod-`.

#### Try to access dev secret (should fail)

```bash
curl http://localhost:8081/v1/projects/test-project/secrets/db-password/versions/latest:access \
  -H "X-Emulator-Principal: serviceAccount:ci@test-project.iam.gserviceaccount.com"
```

**Expected:** `403 Forbidden`

**Why:** The condition `resource.name.startsWith("projects/test-project/secrets/prod-")` evaluates to false for `db-password`

#### Create a production secret

```bash
curl -X POST http://localhost:8081/v1/projects/test-project/secrets?secretId=prod-api-key \
  -H "X-Emulator-Principal: user:alice@example.com" \
  -H "Content-Type: application/json" \
  -d '{
    "replication": {"automatic": {}}
  }'

curl -X POST http://localhost:8081/v1/projects/test-project/secrets/prod-api-key:addVersion \
  -H "X-Emulator-Principal: user:alice@example.com" \
  -H "Content-Type: application/json" \
  -d '{
    "payload": {"data": "'$(echo -n "prod-secret-value" | base64)'"}
  }'
```

#### Access production secret as CI (should succeed)

```bash
curl http://localhost:8081/v1/projects/test-project/secrets/prod-api-key/versions/latest:access \
  -H "X-Emulator-Principal: serviceAccount:ci@test-project.iam.gserviceaccount.com"
```

**Expected:** Success (200 OK)

**Why:** The resource name starts with `projects/test-project/secrets/prod-`, so the condition passes

---

## Part 6: Testing KMS with IAM

### Scenario: Alice encrypts data with KMS

#### Create a key ring

```bash
curl -X POST http://localhost:8082/v1/projects/test-project/locations/global/keyRings?keyRingId=app-keys \
  -H "X-Emulator-Principal: user:alice@example.com" \
  -H "Content-Type: application/json" \
  -d '{}'
```

#### Create a crypto key

```bash
curl -X POST http://localhost:8082/v1/projects/test-project/locations/global/keyRings/app-keys/cryptoKeys?cryptoKeyId=payment-encryption \
  -H "X-Emulator-Principal: user:alice@example.com" \
  -H "Content-Type: application/json" \
  -d '{
    "purpose": "ENCRYPT_DECRYPT"
  }'
```

#### Encrypt some data

```bash
curl -X POST http://localhost:8082/v1/projects/test-project/locations/global/keyRings/app-keys/cryptoKeys/payment-encryption:encrypt \
  -H "X-Emulator-Principal: user:alice@example.com" \
  -H "Content-Type: application/json" \
  -d '{
    "plaintext": "'$(echo -n "credit-card-number" | base64)'"
  }'
```

**Expected:** Returns `{"ciphertext": "...base64..."}`

**Why:** Alice has `cloudkms.cryptoKeys.encrypt` permission

#### Decrypt the data

Save the ciphertext from the previous response, then:

```bash
curl -X POST http://localhost:8082/v1/projects/test-project/locations/global/keyRings/app-keys/cryptoKeys/payment-encryption:decrypt \
  -H "X-Emulator-Principal: user:alice@example.com" \
  -H "Content-Type: application/json" \
  -d '{
    "ciphertext": "BASE64_CIPHERTEXT_HERE"
  }'
```

**Expected:** Returns `{"plaintext": "...base64..."}`, which decodes to `credit-card-number`

---

## Part 7: Multi-Service Integration

### Scenario: Store encrypted secret in Secret Manager

Combine both services: encrypt sensitive data with KMS, then store the ciphertext in Secret Manager.

#### 1. Encrypt credit card with KMS

```bash
CIPHERTEXT=$(curl -s -X POST http://localhost:8082/v1/projects/test-project/locations/global/keyRings/app-keys/cryptoKeys/payment-encryption:encrypt \
  -H "X-Emulator-Principal: user:alice@example.com" \
  -H "Content-Type: application/json" \
  -d '{
    "plaintext": "'$(echo -n "4111-1111-1111-1111" | base64)'"
  }' | jq -r '.ciphertext')

echo "Ciphertext: $CIPHERTEXT"
```

#### 2. Store ciphertext in Secret Manager

```bash
curl -X POST http://localhost:8081/v1/projects/test-project/secrets?secretId=encrypted-cc \
  -H "X-Emulator-Principal: user:alice@example.com" \
  -H "Content-Type: application/json" \
  -d '{
    "replication": {"automatic": {}},
    "labels": {"encrypted-with": "kms"}
  }'

curl -X POST http://localhost:8081/v1/projects/test-project/secrets/encrypted-cc:addVersion \
  -H "X-Emulator-Principal: user:alice@example.com" \
  -H "Content-Type: application/json" \
  -d '{
    "payload": {"data": "'"$CIPHERTEXT"'"}
  }'
```

#### 3. Retrieve and decrypt

```bash
# Retrieve ciphertext from Secret Manager
STORED_CIPHERTEXT=$(curl -s http://localhost:8081/v1/projects/test-project/secrets/encrypted-cc/versions/latest:access \
  -H "X-Emulator-Principal: user:alice@example.com" \
  | jq -r '.payload.data')

# Decrypt with KMS
curl -X POST http://localhost:8082/v1/projects/test-project/locations/global/keyRings/app-keys/cryptoKeys/payment-encryption:decrypt \
  -H "X-Emulator-Principal: user:alice@example.com" \
  -H "Content-Type: application/json" \
  -d '{
    "ciphertext": "'"$STORED_CIPHERTEXT"'"
  }' | jq -r '.plaintext' | base64 -d
```

**Expected output:** `4111-1111-1111-1111`

**What happened:**
1. KMS encrypted the plaintext → ciphertext (requires `cloudkms.cryptoKeys.encrypt`)
2. Secret Manager stored the ciphertext (requires `secretmanager.versions.add`)
3. Secret Manager retrieved the ciphertext (requires `secretmanager.versions.access`)
4. KMS decrypted the ciphertext → plaintext (requires `cloudkms.cryptoKeys.decrypt`)

All four permission checks succeeded because Alice has the `roles/custom.developer` role.

---

## Part 8: Switching IAM Modes

The emulators support three IAM modes:

| Mode | Behavior | Use Case |
|------|----------|----------|
| `off` | No permission checks (legacy) | Local dev, quick prototyping |
| `permissive` | Check permissions, fail-open on errors | Integration tests |
| `strict` | Check permissions, fail-closed on errors | CI/CD |

### Test: Disable IAM entirely

Edit `docker-compose.yml`:

```yaml
secret-manager:
  environment:
    - IAM_MODE=off  # Changed from permissive
```

Restart:
```bash
docker compose restart secret-manager
```

Now try accessing as an unauthorized user:
```bash
curl http://localhost:8081/v1/projects/test-project/secrets/db-password/versions/latest:access \
  -H "X-Emulator-Principal: user:charlie@example.com"
```

**Expected:** Success (200 OK) - permission checks are disabled

### Test: Strict mode

Edit `docker-compose.yml`:

```yaml
secret-manager:
  environment:
    - IAM_MODE=strict
```

Restart and test. In strict mode, any IAM connectivity issue results in denial (fail-closed).

---

## Part 9: Using the Go SDK

For applications using the official GCP Go SDK:

```go
package main

import (
    "context"
    "fmt"
    "log"

    secretmanager "cloud.google.com/go/secretmanager/apiv1"
    "cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
    "google.golang.org/api/option"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
    "google.golang.org/grpc/metadata"
)

func main() {
    ctx := context.Background()

    // Connect to emulator
    conn, err := grpc.NewClient(
        "localhost:9090",
        grpc.WithTransportCredentials(insecure.NewCredentials()),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer conn.Close()

    client, err := secretmanager.NewClient(ctx, option.WithGRPCConn(conn))
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // Inject principal identity
    ctx = metadata.AppendToOutgoingContext(ctx, "x-emulator-principal", "user:alice@example.com")

    // Access secret
    resp, err := client.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{
        Name: "projects/test-project/secrets/db-password/versions/latest",
    })
    if err != nil {
        log.Fatalf("Failed to access secret: %v", err)
    }

    fmt.Printf("Secret value: %s\n", string(resp.Payload.Data))
}
```

Run:
```bash
go run main.go
```

**Output:** `Secret value: super-secret-password`

---

## Part 10: Debugging Permission Issues

### Enable trace logging

Edit `docker-compose.yml`:

```yaml
iam:
  command: ["--config", "/policy.yaml", "--trace"]
```

Restart:
```bash
docker compose restart iam
```

### Watch IAM logs

```bash
docker compose logs -f iam
```

You'll see detailed permission check logs:
```
[TRACE] CheckPermission: principal=user:alice@example.com resource=projects/test-project/secrets/db-password permission=secretmanager.versions.access result=ALLOW
```

### Common issues

**1. Permission denied but user is in correct role**

Check:
- Resource name format (must match canonical form)
- Condition expression (may be blocking access)
- Group membership (user must be in group)

**2. IAM emulator not reachable**

Check:
- `IAM_HOST` environment variable points to correct service
- IAM emulator health check passing
- Data plane emulator started after IAM (use `depends_on`)

**3. Principal not propagated**

Check:
- Header name: `x-emulator-principal` (gRPC) or `X-Emulator-Principal` (HTTP)
- Header value format: `user:email`, `serviceAccount:email`, or `group:name`

---

## Part 11: CI/CD Integration

### GitHub Actions Example

`.github/workflows/test.yml`:

```yaml
name: Integration Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    
    services:
      emulators:
        image: ghcr.io/blackwell-systems/gcp-emulator-control-plane:latest
        ports:
          - 8080:8080  # IAM
          - 9090:9090  # Secret Manager gRPC
          - 8081:8080  # Secret Manager HTTP
          - 9091:9090  # KMS gRPC
          - 8082:8080  # KMS HTTP
        options: >-
          --health-cmd "curl -f http://localhost:8080/health"
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5

    steps:
      - uses: actions/checkout@v4
      
      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.24'
      
      - name: Run integration tests
        run: go test ./... -tags=integration
        env:
          SECRET_MANAGER_HOST: localhost:9090
          KMS_HOST: localhost:9091
          TEST_PRINCIPAL: serviceAccount:ci@test-project.iam.gserviceaccount.com
```

### Docker Compose in CI

```bash
# Start stack
docker compose up -d

# Wait for health checks
docker compose ps

# Run tests
go test ./... -tags=integration

# Cleanup
docker compose down
```

---

## Summary

You now have a complete local GCP environment with:

+ IAM-enforced authorization across multiple services
+ Realistic permission checks that mirror production GCP
+ Conditional bindings for fine-grained access control
+ Multi-service integration (Secret Manager + KMS)
+ CI-ready setup with Docker Compose

**Next steps:**
- Customize `policy.yaml` for your application's needs
- Add more emulators as they become available
- Integrate with your existing test suite
- Use strict mode in CI to catch permission issues early

**Resources:**
- [Integration Contract](INTEGRATION_CONTRACT.md) - How to add new emulators
- [Policy Examples](../packs/) - Ready-to-use role definitions
- [Troubleshooting](TROUBLESHOOTING.md) - Common issues and solutions
