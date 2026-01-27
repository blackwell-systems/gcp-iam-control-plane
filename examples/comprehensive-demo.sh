#!/bin/bash
# Comprehensive demo of GCP Emulator Control Plane
# Exercises CLI, Secret Manager, KMS, and IAM enforcement

set -e

PRINCIPAL="user:alice@example.com"
UNAUTHORIZED_PRINCIPAL="user:unauthorized@example.com"
SM_URL="http://localhost:8081"
KMS_URL="http://localhost:8082"
PROJECT="test-project"

echo "=========================================="
echo "GCP Emulator Control Plane - Full Demo"
echo "=========================================="
echo ""

# Check if gcp-emulator is available
if ! command -v gcp-emulator &> /dev/null; then
    echo "Error: gcp-emulator CLI not found. Build it first:"
    echo "  go build -o gcp-emulator ./cmd/gcp-emulator"
    exit 1
fi

# Start the stack
echo "=== Starting GCP Emulator Stack ==="
gcp-emulator start --mode=permissive
echo "Waiting for services to be ready..."
sleep 10
echo ""

# Check status
echo "=== Stack Status ==="
gcp-emulator status
echo ""

# ==========================================
# Secret Manager Demo
# ==========================================
echo "=========================================="
echo "Secret Manager Operations"
echo "=========================================="
echo ""

# Create multiple secrets
echo "1. Creating secrets..."
for secret in db-password api-key service-token; do
    echo "  Creating secret: $secret"
    curl -s -X POST "$SM_URL/v1/projects/$PROJECT/secrets?secretId=$secret" \
        -H "X-Emulator-Principal: $PRINCIPAL" \
        -H "Content-Type: application/json" \
        -d '{"replication":{"automatic":{}}}' > /dev/null
done
echo ""

# Add versions to secrets
echo "2. Adding secret versions..."
for secret in db-password api-key service-token; do
    echo "  Adding version to: $secret"
    SECRET_DATA=$(echo -n "$secret-value-v1" | base64)
    curl -s -X POST "$SM_URL/v1/projects/$PROJECT/secrets/$secret:addVersion" \
        -H "X-Emulator-Principal: $PRINCIPAL" \
        -H "Content-Type: application/json" \
        -d "{\"payload\":{\"data\":\"$SECRET_DATA\"}}" > /dev/null
done
echo ""

# List all secrets
echo "3. Listing all secrets..."
curl -s "$SM_URL/v1/projects/$PROJECT/secrets" \
    -H "X-Emulator-Principal: $PRINCIPAL" | jq -r '.secrets[].name'
echo ""

# Access a secret
echo "4. Accessing secret 'db-password'..."
RESPONSE=$(curl -s "$SM_URL/v1/projects/$PROJECT/secrets/db-password/versions/1:access" \
    -H "X-Emulator-Principal: $PRINCIPAL")
SECRET_VALUE=$(echo "$RESPONSE" | jq -r '.payload.data' | base64 -d)
echo "  Secret value: $SECRET_VALUE"
echo ""

# Add second version
echo "5. Adding second version to 'db-password'..."
SECRET_DATA=$(echo -n "db-password-value-v2" | base64)
curl -s -X POST "$SM_URL/v1/projects/$PROJECT/secrets/db-password:addVersion" \
    -H "X-Emulator-Principal: $PRINCIPAL" \
    -H "Content-Type: application/json" \
    -d "{\"payload\":{\"data\":\"$SECRET_DATA\"}}" > /dev/null
echo ""

# List versions
echo "6. Listing versions of 'db-password'..."
curl -s "$SM_URL/v1/projects/$PROJECT/secrets/db-password/versions" \
    -H "X-Emulator-Principal: $PRINCIPAL" | jq -r '.versions[] | "\(.name) - \(.state)"'
echo ""

# Access latest version
echo "7. Accessing latest version..."
RESPONSE=$(curl -s "$SM_URL/v1/projects/$PROJECT/secrets/db-password/versions/latest:access" \
    -H "X-Emulator-Principal: $PRINCIPAL")
SECRET_VALUE=$(echo "$RESPONSE" | jq -r '.payload.data' | base64 -d)
echo "  Latest secret value: $SECRET_VALUE"
echo ""

# Disable version 1
echo "8. Disabling version 1..."
curl -s -X POST "$SM_URL/v1/projects/$PROJECT/secrets/db-password/versions/1:disable" \
    -H "X-Emulator-Principal: $PRINCIPAL" > /dev/null
echo "  Version 1 disabled"
echo ""

# Try to access disabled version (should fail)
echo "9. Attempting to access disabled version 1..."
RESPONSE=$(curl -s -w "\n%{http_code}" "$SM_URL/v1/projects/$PROJECT/secrets/db-password/versions/1:access" \
    -H "X-Emulator-Principal: $PRINCIPAL")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
if [ "$HTTP_CODE" != "200" ]; then
    echo "  ✓ Correctly denied access to disabled version (HTTP $HTTP_CODE)"
else
    echo "  ✗ Should have denied access to disabled version"
fi
echo ""

# ==========================================
# KMS Demo
# ==========================================
echo "=========================================="
echo "KMS Operations"
echo "=========================================="
echo ""

# Create key ring
echo "1. Creating key ring 'app-keys'..."
curl -s -X POST "$KMS_URL/v1/projects/$PROJECT/locations/global/keyRings?keyRingId=app-keys" \
    -H "X-Emulator-Principal: $PRINCIPAL" \
    -H "Content-Type: application/json" > /dev/null
echo ""

# Create crypto keys
echo "2. Creating crypto keys..."
for key in encryption-key signing-key backup-key; do
    echo "  Creating key: $key"
    curl -s -X POST "$KMS_URL/v1/projects/$PROJECT/locations/global/keyRings/app-keys/cryptoKeys?cryptoKeyId=$key" \
        -H "X-Emulator-Principal: $PRINCIPAL" \
        -H "Content-Type: application/json" \
        -d '{"purpose":"ENCRYPT_DECRYPT"}' > /dev/null
done
echo ""

# List keys
echo "3. Listing crypto keys..."
curl -s "$KMS_URL/v1/projects/$PROJECT/locations/global/keyRings/app-keys/cryptoKeys" \
    -H "X-Emulator-Principal: $PRINCIPAL" | jq -r '.cryptoKeys[].name'
echo ""

# Encrypt data
echo "4. Encrypting sensitive data with 'encryption-key'..."
PLAINTEXT=$(echo -n "credit-card-number: 4111-1111-1111-1111" | base64)
ENCRYPT_RESPONSE=$(curl -s -X POST "$KMS_URL/v1/projects/$PROJECT/locations/global/keyRings/app-keys/cryptoKeys/encryption-key:encrypt" \
    -H "X-Emulator-Principal: $PRINCIPAL" \
    -H "Content-Type: application/json" \
    -d "{\"plaintext\":\"$PLAINTEXT\"}")
CIPHERTEXT=$(echo "$ENCRYPT_RESPONSE" | jq -r '.ciphertext')
echo "  ✓ Data encrypted (ciphertext: ${CIPHERTEXT:0:40}...)"
echo ""

# Decrypt data
echo "5. Decrypting data..."
DECRYPT_RESPONSE=$(curl -s -X POST "$KMS_URL/v1/projects/$PROJECT/locations/global/keyRings/app-keys/cryptoKeys/encryption-key:decrypt" \
    -H "X-Emulator-Principal: $PRINCIPAL" \
    -H "Content-Type: application/json" \
    -d "{\"ciphertext\":\"$CIPHERTEXT\"}")
DECRYPTED_PLAINTEXT=$(echo "$DECRYPT_RESPONSE" | jq -r '.plaintext' | base64 -d)
echo "  ✓ Data decrypted: $DECRYPTED_PLAINTEXT"
echo ""

# Encrypt multiple pieces of data
echo "6. Encrypting multiple data items..."
for item in "user-password-123" "api-token-xyz" "database-connection-string"; do
    echo "  Encrypting: $item"
    PLAINTEXT=$(echo -n "$item" | base64)
    curl -s -X POST "$KMS_URL/v1/projects/$PROJECT/locations/global/keyRings/app-keys/cryptoKeys/encryption-key:encrypt" \
        -H "X-Emulator-Principal: $PRINCIPAL" \
        -H "Content-Type: application/json" \
        -d "{\"plaintext\":\"$PLAINTEXT\"}" > /dev/null
done
echo ""

# ==========================================
# IAM Enforcement Demo
# ==========================================
echo "=========================================="
echo "IAM Enforcement"
echo "=========================================="
echo ""

# Authorized user operations
echo "1. Testing authorized principal ($PRINCIPAL)..."
RESPONSE=$(curl -s -w "\n%{http_code}" "$SM_URL/v1/projects/$PROJECT/secrets" \
    -H "X-Emulator-Principal: $PRINCIPAL")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
if [ "$HTTP_CODE" = "200" ]; then
    echo "  ✓ Authorized access granted (HTTP $HTTP_CODE)"
else
    echo "  ✗ Authorized access denied (HTTP $HTTP_CODE)"
fi
echo ""

# Unauthorized user operations
echo "2. Testing unauthorized principal ($UNAUTHORIZED_PRINCIPAL)..."
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$SM_URL/v1/projects/$PROJECT/secrets?secretId=unauthorized-secret" \
    -H "X-Emulator-Principal: $UNAUTHORIZED_PRINCIPAL" \
    -H "Content-Type: application/json" \
    -d '{"replication":{"automatic":{}}}')
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
if [ "$HTTP_CODE" = "403" ]; then
    echo "  ✓ Unauthorized access correctly denied (HTTP $HTTP_CODE)"
else
    echo "  ✗ Should have denied unauthorized access (HTTP $HTTP_CODE)"
fi
echo ""

# Missing principal
echo "3. Testing request without principal header..."
RESPONSE=$(curl -s -w "\n%{http_code}" "$SM_URL/v1/projects/$PROJECT/secrets")
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
if [ "$HTTP_CODE" = "403" ]; then
    echo "  ✓ Request without principal correctly denied (HTTP $HTTP_CODE)"
else
    echo "  Note: Request without principal returned HTTP $HTTP_CODE"
fi
echo ""

# ==========================================
# Policy Validation
# ==========================================
echo "=========================================="
echo "Policy Validation"
echo "=========================================="
echo ""

echo "1. Validating policy.yaml..."
gcp-emulator policy validate
echo ""

# ==========================================
# Integration Example: Secret-Protected Encryption
# ==========================================
echo "=========================================="
echo "Integration: Secret-Protected Encryption"
echo "=========================================="
echo ""

echo "Scenario: Store encryption key in Secret Manager, use it with KMS"
echo ""

# Store encryption key reference in Secret Manager
echo "1. Storing key reference in Secret Manager..."
KEY_REF="projects/$PROJECT/locations/global/keyRings/app-keys/cryptoKeys/encryption-key"
KEY_REF_DATA=$(echo -n "$KEY_REF" | base64)
curl -s -X POST "$SM_URL/v1/projects/$PROJECT/secrets?secretId=kms-key-ref" \
    -H "X-Emulator-Principal: $PRINCIPAL" \
    -H "Content-Type: application/json" \
    -d '{"replication":{"automatic":{}}}' > /dev/null

curl -s -X POST "$SM_URL/v1/projects/$PROJECT/secrets/kms-key-ref:addVersion" \
    -H "X-Emulator-Principal: $PRINCIPAL" \
    -H "Content-Type: application/json" \
    -d "{\"payload\":{\"data\":\"$KEY_REF_DATA\"}}" > /dev/null
echo "  ✓ Key reference stored"
echo ""

# Retrieve key reference
echo "2. Retrieving key reference from Secret Manager..."
RESPONSE=$(curl -s "$SM_URL/v1/projects/$PROJECT/secrets/kms-key-ref/versions/latest:access" \
    -H "X-Emulator-Principal: $PRINCIPAL")
RETRIEVED_KEY_REF=$(echo "$RESPONSE" | jq -r '.payload.data' | base64 -d)
echo "  Retrieved: $RETRIEVED_KEY_REF"
echo ""

# Use retrieved key for encryption
echo "3. Using retrieved key to encrypt data..."
DATA_TO_ENCRYPT=$(echo -n "integrated-secret-data" | base64)
ENCRYPT_RESPONSE=$(curl -s -X POST "$KMS_URL/v1/$RETRIEVED_KEY_REF:encrypt" \
    -H "X-Emulator-Principal: $PRINCIPAL" \
    -H "Content-Type: application/json" \
    -d "{\"plaintext\":\"$DATA_TO_ENCRYPT\"}")
INTEGRATED_CIPHERTEXT=$(echo "$ENCRYPT_RESPONSE" | jq -r '.ciphertext')
echo "  ✓ Data encrypted using key from Secret Manager"
echo ""

# ==========================================
# Cleanup and Summary
# ==========================================
echo "=========================================="
echo "Cleanup"
echo "=========================================="
echo ""

echo "Stopping emulator stack..."
gcp-emulator stop
echo ""

echo "=========================================="
echo "Demo Complete!"
echo "=========================================="
echo ""
echo "Summary of operations performed:"
echo "  - Created 3 secrets with multiple versions"
echo "  - Demonstrated version management (disable, access)"
echo "  - Created 1 key ring with 3 crypto keys"
echo "  - Performed encryption/decryption operations"
echo "  - Tested IAM enforcement (authorized, unauthorized, missing principal)"
echo "  - Validated policy configuration"
echo "  - Demonstrated Secret Manager + KMS integration"
echo ""
echo "All operations performed with:"
echo "  - IAM Mode: permissive"
echo "  - Authorized Principal: $PRINCIPAL"
echo "  - Project: $PROJECT"
echo ""
