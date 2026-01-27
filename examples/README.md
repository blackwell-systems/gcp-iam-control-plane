# Examples

This directory contains practical examples demonstrating the GCP Emulator Control Plane.

## Quick Start Examples

### Comprehensive Demo

**`comprehensive-demo.sh`** - Complete walkthrough of all major features:

```bash
# Start stack, run full demo, stop stack
./examples/comprehensive-demo.sh
```

**What it demonstrates:**
- CLI commands (start, stop, status, policy validate)
- Secret Manager operations:
  - Creating multiple secrets
  - Adding and managing versions
  - Listing secrets and versions
  - Accessing latest/specific versions
  - Disabling versions (with access denial verification)
- KMS operations:
  - Creating key rings and crypto keys
  - Encrypting and decrypting data
  - Multiple encryption operations
- IAM enforcement:
  - Authorized principal access
  - Unauthorized principal denial
  - Missing principal handling
- Integration pattern:
  - Storing KMS key references in Secret Manager
  - Retrieving and using keys for encryption

**Output:** Comprehensive test of all services with clear status indicators.

---

## Protocol-Specific Examples

### REST API (curl)

Located in `curl/`:

**`test-secret-manager.sh`** - Secret Manager REST API
- Create secret
- Add version
- Access version
- List secrets

**`test-kms.sh`** - KMS REST API
- Create key ring
- Create crypto key
- Encrypt data
- Decrypt data

### gRPC (Go SDK)

Located in `go/`:

**`main.go`** - Go SDK integration
- Connect to emulators
- Use official GCP client libraries
- Handle IAM principal injection
- Full Secret Manager and KMS workflows

---

## Running Examples

### Prerequisites

```bash
# Build CLI
go build -o gcp-emulator ./cmd/gcp-emulator

# Add to PATH or use ./gcp-emulator
export PATH=$PATH:$(pwd)
```

### Run Comprehensive Demo

```bash
# Automatic (starts and stops stack)
./examples/comprehensive-demo.sh
```

### Run Individual Examples

```bash
# Start stack first
gcp-emulator start --mode=permissive

# Run REST examples
./examples/curl/test-secret-manager.sh
./examples/curl/test-kms.sh

# Run Go example
cd examples/go
go run main.go

# Stop stack when done
gcp-emulator stop
```

---

## Example Principals

All examples use principals defined in `policy.yaml`:

| Principal | Role | Permissions |
|-----------|------|-------------|
| `user:alice@example.com` | `roles/custom.admin` | Full access to secrets and keys |
| `user:bob@example.com` | `roles/custom.developer` | Read-only access |
| `user:unauthorized@example.com` | None | No access (used for testing denials) |

---

## IAM Modes

Examples demonstrate different IAM enforcement modes:

| Mode | Behavior | Use Case |
|------|----------|----------|
| `off` | No permission checks | Legacy compatibility |
| `permissive` | Check permissions, fail-open | Local development |
| `strict` | Check permissions, fail-closed | CI/CD pipelines |

**Comprehensive demo uses `permissive` mode.**

---

## REST API Ports

| Service | gRPC Port | HTTP Port | Health Port |
|---------|-----------|-----------|-------------|
| IAM Emulator | 8080 | N/A | 9080 |
| Secret Manager | 9090 | 8081 | 8081 |
| KMS | 9091 | 8082 | 8082 |

---

## Common Patterns

### Principal Injection

**REST (curl):**
```bash
curl -H "X-Emulator-Principal: user:alice@example.com" \
  http://localhost:8081/v1/projects/test-project/secrets
```

**gRPC (Go):**
```go
ctx = metadata.AppendToOutgoingContext(ctx, 
    "x-emulator-principal", "user:alice@example.com")
```

### Base64 Encoding

Secret Manager requires base64-encoded payloads:

```bash
# Encode
echo -n "my-secret-value" | base64

# Decode
echo "bXktc2VjcmV0LXZhbHVl" | base64 -d
```

### Error Handling

Check HTTP status codes:

```bash
RESPONSE=$(curl -s -w "\n%{http_code}" http://localhost:8081/...)
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | sed '$d')

if [ "$HTTP_CODE" = "200" ]; then
    echo "Success"
else
    echo "Error: HTTP $HTTP_CODE"
    echo "$BODY" | jq .
fi
```

---

## Next Steps

- Read [ARCHITECTURE.md](../docs/ARCHITECTURE.md) for system design
- Review [END_TO_END_TUTORIAL.md](../docs/END_TO_END_TUTORIAL.md) for detailed walkthrough
- Check [TROUBLESHOOTING.md](../docs/TROUBLESHOOTING.md) if you encounter issues
- See [MIGRATION.md](../docs/MIGRATION.md) for migrating from standalone emulators

---

## Contributing Examples

Have a useful example pattern? Contributions welcome!

1. Add to appropriate directory (`curl/`, `go/`, or root)
2. Include clear comments and error handling
3. Test with all IAM modes
4. Update this README
5. Submit PR
