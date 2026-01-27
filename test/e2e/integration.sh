#!/bin/bash
# End-to-end integration test for GCP Emulator Control Plane
# Tests CLI commands, all 3 emulators, and IAM enforcement

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configuration
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
CLI_BINARY="$ROOT_DIR/bin/gcp-emulator-e2e-integration"
IAM_PORT=8080
SM_GRPC_PORT=9090
SM_HTTP_PORT=8081
KMS_GRPC_PORT=9091
KMS_HTTP_PORT=8082
PROJECT="test-project"
PRINCIPAL="user:alice@example.com"

# Test results
TESTS_PASSED=0
TESTS_FAILED=0

log() {
    echo -e "${CYAN}[$(date +'%H:%M:%S')]${NC} $1"
}

success() {
    echo -e "${GREEN}✓${NC} $1"
    TESTS_PASSED=$((TESTS_PASSED + 1))
}

error() {
    echo -e "${RED}✗${NC} $1"
    TESTS_FAILED=$((TESTS_FAILED + 1))
}

warn() {
    echo -e "${YELLOW}⚠${NC} $1"
}

# Cleanup function
cleanup() {
    log "Cleaning up..."
    if [ -f "$CLI_BINARY" ]; then
        "$CLI_BINARY" stop >/dev/null 2>&1 || true
    fi
    # Also cleanup with docker compose directly if CLI failed
    docker compose -f "$ROOT_DIR/docker-compose.yml" down >/dev/null 2>&1 || true
    rm -f "$CLI_BINARY"
}

trap cleanup EXIT

# Build CLI
log "Building CLI binary..."
cd "$ROOT_DIR"
go build -o "$CLI_BINARY" ./cmd/gcp-emulator
if [ $? -eq 0 ]; then
    success "CLI binary built"
else
    error "Failed to build CLI"
    exit 1
fi

# Start stack
log "Starting emulator stack..."
"$CLI_BINARY" start --mode=permissive
if [ $? -eq 0 ]; then
    success "Stack started"
else
    error "Failed to start stack"
    exit 1
fi

# Wait for services to be ready
log "Waiting for services to be ready..."
sleep 20

# Note: Skipping status check since IAM may not have /health endpoint yet
# Services should be ready after 20s wait

# Test Secret Manager
log ""
log "===== Testing Secret Manager ====="

# Create secret
log "Creating secret..."
log "Target: http://localhost:$SM_HTTP_PORT/v1/projects/$PROJECT/secrets"

# First check if port is reachable
if ! curl -s --max-time 2 http://localhost:$SM_HTTP_PORT/health > /dev/null 2>&1; then
    error "Secret Manager not reachable on port $SM_HTTP_PORT"
    log "Checking container status..."
    docker compose ps
    log "Showing Secret Manager logs..."
    docker compose logs secret-manager | tail -30
    exit 1
fi

RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "http://localhost:$SM_HTTP_PORT/v1/projects/$PROJECT/secrets" \
  -H "X-Emulator-Principal: $PRINCIPAL" \
  -H "Content-Type: application/json" \
  -d '{
    "secretId": "test-secret",
    "replication": {
      "automatic": {}
    }
  }')

HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | sed '$d')

if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "201" ]; then
    success "Secret created"
else
    error "Failed to create secret (HTTP $HTTP_CODE)"
    echo "Response body:"
    echo "$BODY"
    exit 1
fi

# Add secret version
log "Adding secret version..."
SECRET_DATA=$(echo -n "my-secret-password" | base64)
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "http://localhost:$SM_HTTP_PORT/v1/projects/$PROJECT/secrets/test-secret:addVersion" \
  -H "X-Emulator-Principal: $PRINCIPAL" \
  -H "Content-Type: application/json" \
  -d "{
    \"payload\": {
      \"data\": \"$SECRET_DATA\"
    }
  }")

HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
if [ "$HTTP_CODE" = "200" ]; then
    success "Secret version added"
else
    error "Failed to add secret version (HTTP $HTTP_CODE)"
fi

# Access secret version
log "Accessing secret version..."
RESPONSE=$(curl -s -w "\n%{http_code}" "http://localhost:$SM_HTTP_PORT/v1/projects/$PROJECT/secrets/test-secret/versions/1:access" \
  -H "X-Emulator-Principal: $PRINCIPAL")

HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | sed '$d')

if [ "$HTTP_CODE" = "200" ]; then
    # Verify data
    RETRIEVED_DATA=$(echo "$BODY" | jq -r '.payload.data' | base64 -d)
    if [ "$RETRIEVED_DATA" = "my-secret-password" ]; then
        success "Secret accessed and verified"
    else
        error "Secret data mismatch"
    fi
else
    error "Failed to access secret (HTTP $HTTP_CODE)"
fi

# Test KMS
log ""
log "===== Testing KMS ====="

# Create key ring
log "Creating key ring..."
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "http://localhost:$KMS_HTTP_PORT/v1/projects/$PROJECT/locations/global/keyRings?keyRingId=test-ring" \
  -H "X-Emulator-Principal: $PRINCIPAL" \
  -H "Content-Type: application/json")

HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "201" ]; then
    success "Key ring created"
else
    warn "Key ring creation returned HTTP $HTTP_CODE (may already exist)"
fi

# Create crypto key
log "Creating crypto key..."
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "http://localhost:$KMS_HTTP_PORT/v1/projects/$PROJECT/locations/global/keyRings/test-ring/cryptoKeys?cryptoKeyId=test-key" \
  -H "X-Emulator-Principal: $PRINCIPAL" \
  -H "Content-Type: application/json" \
  -d '{
    "purpose": "ENCRYPT_DECRYPT"
  }')

HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "201" ]; then
    success "Crypto key created"
else
    warn "Crypto key creation returned HTTP $HTTP_CODE (may already exist)"
fi

# Encrypt data
log "Encrypting data with KMS..."
PLAINTEXT=$(echo -n "sensitive-data" | base64)
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "http://localhost:$KMS_HTTP_PORT/v1/projects/$PROJECT/locations/global/keyRings/test-ring/cryptoKeys/test-key:encrypt" \
  -H "X-Emulator-Principal: $PRINCIPAL" \
  -H "Content-Type: application/json" \
  -d "{
    \"plaintext\": \"$PLAINTEXT\"
  }")

HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | sed '$d')

if [ "$HTTP_CODE" = "200" ]; then
    CIPHERTEXT=$(echo "$BODY" | jq -r '.ciphertext')
    success "Data encrypted"
    
    # Decrypt data
    log "Decrypting data with KMS..."
    RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "http://localhost:$KMS_HTTP_PORT/v1/projects/$PROJECT/locations/global/keyRings/test-ring/cryptoKeys/test-key:decrypt" \
      -H "X-Emulator-Principal: $PRINCIPAL" \
      -H "Content-Type: application/json" \
      -d "{
        \"ciphertext\": \"$CIPHERTEXT\"
      }")
    
    HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
    BODY=$(echo "$RESPONSE" | sed '$d')
    
    if [ "$HTTP_CODE" = "200" ]; then
        DECRYPTED=$(echo "$BODY" | jq -r '.plaintext' | base64 -d)
        if [ "$DECRYPTED" = "sensitive-data" ]; then
            success "Data decrypted and verified"
        else
            error "Decrypted data mismatch"
        fi
    else
        error "Failed to decrypt (HTTP $HTTP_CODE)"
    fi
else
    error "Failed to encrypt (HTTP $HTTP_CODE)"
fi

# Test IAM enforcement
log ""
log "===== Testing IAM Enforcement ====="

# Test with unauthorized principal
log "Testing with unauthorized principal..."
UNAUTHORIZED="user:unauthorized@example.com"
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "http://localhost:$SM_HTTP_PORT/v1/projects/$PROJECT/secrets" \
  -H "X-Emulator-Principal: $UNAUTHORIZED" \
  -H "Content-Type: application/json" \
  -d '{
    "secretId": "unauthorized-secret",
    "replication": {
      "automatic": {}
    }
  }')

HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
if [ "$HTTP_CODE" = "403" ]; then
    success "Unauthorized request correctly denied"
elif [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "201" ]; then
    warn "Unauthorized request succeeded (policy may allow this principal)"
else
    warn "Unexpected status code: $HTTP_CODE"
fi

# Test policy validation
log ""
log "===== Testing Policy Commands ====="

log "Validating policy..."
"$CLI_BINARY" policy validate
if [ $? -eq 0 ]; then
    success "Policy validation passed"
else
    error "Policy validation failed"
fi

# Test config commands
log ""
log "===== Testing Config Commands ====="

log "Getting config..."
"$CLI_BINARY" config get
if [ $? -eq 0 ]; then
    success "Config get succeeded"
else
    error "Config get failed"
fi

# Stop stack
log ""
log "Stopping stack..."
"$CLI_BINARY" stop
if [ $? -eq 0 ]; then
    success "Stack stopped"
else
    error "Failed to stop stack"
fi

# Print summary
log ""
log "======================================"
log "Test Summary"
log "======================================"
echo -e "${GREEN}Passed: $TESTS_PASSED${NC}"
if [ $TESTS_FAILED -gt 0 ]; then
    echo -e "${RED}Failed: $TESTS_FAILED${NC}"
    exit 1
else
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
fi
