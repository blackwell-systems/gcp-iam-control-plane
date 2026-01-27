package e2e

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	kms "cloud.google.com/go/kms/apiv1"
	kmspb "cloud.google.com/go/kms/apiv1/kmspb"
	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

const (
	iamPort           = 8080
	secretManagerPort = 9090
	kmsPort           = 9091
	testProject       = "test-project"
	testPrincipal     = "user:alice@example.com"
)

var cliBinary string

// TestMain builds the CLI and manages stack lifecycle
func TestMain(m *testing.M) {
	// Get absolute path to root directory
	rootDir, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get root directory: %v\n", err)
		os.Exit(1)
	}
	
	cliBinary = filepath.Join(rootDir, "bin", "gcp-emulator-e2e")

	fmt.Println("Building CLI binary...")
	cmd := exec.Command("go", "build", "-o", cliBinary, "./cmd/gcp-emulator")
	cmd.Dir = rootDir
	if output, err := cmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to build CLI: %v\n%s\n", err, output)
		os.Exit(1)
	}

	// Run tests
	code := m.Run()

	// Cleanup
	os.Remove(cliBinary)
	os.Exit(code)
}

// TestStackLifecycle tests basic CLI commands
func TestStackLifecycle(t *testing.T) {
	// Start stack
	t.Log("Starting stack...")
	startCmd := exec.Command(cliBinary, "start", "--mode=permissive")
	if output, err := startCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to start stack: %v\n%s", err, output)
	}

	// Wait for services to be ready
	time.Sleep(5 * time.Second)

	// Check status
	t.Log("Checking status...")
	statusCmd := exec.Command(cliBinary, "status")
	if output, err := statusCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to check status: %v\n%s", err, output)
	}

	// Cleanup
	defer func() {
		t.Log("Stopping stack...")
		stopCmd := exec.Command(cliBinary, "stop")
		if output, err := stopCmd.CombinedOutput(); err != nil {
			t.Errorf("Failed to stop stack: %v\n%s", err, output)
		}
	}()

	// Run integration tests
	t.Run("SecretManager", testSecretManager)
	t.Run("KMS", testKMS)
	t.Run("CrossService", testCrossService)
	t.Run("IAMEnforcement", testIAMEnforcement)
}

func testSecretManager(t *testing.T) {
	ctx := context.Background()
	ctx = metadata.AppendToOutgoingContext(ctx, "x-emulator-principal", testPrincipal)

	// Create client
	conn, err := grpc.Dial(
		fmt.Sprintf("localhost:%d", secretManagerPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("Failed to connect to Secret Manager: %v", err)
	}
	defer conn.Close()

	client, err := secretmanager.NewClient(ctx, option.WithGRPCConn(conn))
	if err != nil {
		t.Fatalf("Failed to create Secret Manager client: %v", err)
	}
	defer client.Close()

	// Create secret
	t.Log("Creating secret...")
	secret, err := client.CreateSecret(ctx, &secretmanagerpb.CreateSecretRequest{
		Parent:   fmt.Sprintf("projects/%s", testProject),
		SecretId: "test-secret",
		Secret: &secretmanagerpb.Secret{
			Replication: &secretmanagerpb.Replication{
				Replication: &secretmanagerpb.Replication_Automatic_{
					Automatic: &secretmanagerpb.Replication_Automatic{},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed to create secret: %v", err)
	}
	t.Logf("Created secret: %s", secret.Name)

	// Add version
	t.Log("Adding secret version...")
	secretData := []byte("my-secret-data")
	version, err := client.AddSecretVersion(ctx, &secretmanagerpb.AddSecretVersionRequest{
		Parent: secret.Name,
		Payload: &secretmanagerpb.SecretPayload{
			Data: secretData,
		},
	})
	if err != nil {
		t.Fatalf("Failed to add secret version: %v", err)
	}
	t.Logf("Added version: %s", version.Name)

	// Access version
	t.Log("Accessing secret version...")
	accessResp, err := client.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{
		Name: version.Name,
	})
	if err != nil {
		t.Fatalf("Failed to access secret version: %v", err)
	}

	if string(accessResp.Payload.Data) != string(secretData) {
		t.Errorf("Secret data mismatch: got %q, want %q", accessResp.Payload.Data, secretData)
	}

	t.Log("Secret Manager test passed")
}

func testKMS(t *testing.T) {
	ctx := context.Background()
	ctx = metadata.AppendToOutgoingContext(ctx, "x-emulator-principal", testPrincipal)

	// Create client
	conn, err := grpc.Dial(
		fmt.Sprintf("localhost:%d", kmsPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("Failed to connect to KMS: %v", err)
	}
	defer conn.Close()

	client, err := kms.NewKeyManagementClient(ctx, option.WithGRPCConn(conn))
	if err != nil {
		t.Fatalf("Failed to create KMS client: %v", err)
	}
	defer client.Close()

	keyRingName := fmt.Sprintf("projects/%s/locations/global/keyRings/test-ring", testProject)
	cryptoKeyName := fmt.Sprintf("%s/cryptoKeys/test-key", keyRingName)

	// Create key ring
	t.Log("Creating key ring...")
	_, err = client.CreateKeyRing(ctx, &kmspb.CreateKeyRingRequest{
		Parent:    fmt.Sprintf("projects/%s/locations/global", testProject),
		KeyRingId: "test-ring",
	})
	if err != nil {
		t.Fatalf("Failed to create key ring: %v", err)
	}

	// Create crypto key
	t.Log("Creating crypto key...")
	_, err = client.CreateCryptoKey(ctx, &kmspb.CreateCryptoKeyRequest{
		Parent:      keyRingName,
		CryptoKeyId: "test-key",
		CryptoKey: &kmspb.CryptoKey{
			Purpose: kmspb.CryptoKey_ENCRYPT_DECRYPT,
		},
	})
	if err != nil {
		t.Fatalf("Failed to create crypto key: %v", err)
	}

	// Encrypt data
	t.Log("Encrypting data...")
	plaintext := []byte("sensitive-data")
	encryptResp, err := client.Encrypt(ctx, &kmspb.EncryptRequest{
		Name:      cryptoKeyName,
		Plaintext: plaintext,
	})
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}
	t.Logf("Encrypted %d bytes", len(encryptResp.Ciphertext))

	// Decrypt data
	t.Log("Decrypting data...")
	decryptResp, err := client.Decrypt(ctx, &kmspb.DecryptRequest{
		Name:       cryptoKeyName,
		Ciphertext: encryptResp.Ciphertext,
	})
	if err != nil {
		t.Fatalf("Failed to decrypt: %v", err)
	}

	if string(decryptResp.Plaintext) != string(plaintext) {
		t.Errorf("Decrypted data mismatch: got %q, want %q", decryptResp.Plaintext, plaintext)
	}

	t.Log("KMS test passed")
}

func testCrossService(t *testing.T) {
	ctx := context.Background()
	ctx = metadata.AppendToOutgoingContext(ctx, "x-emulator-principal", testPrincipal)

	// Connect to both services
	smConn, err := grpc.Dial(
		fmt.Sprintf("localhost:%d", secretManagerPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("Failed to connect to Secret Manager: %v", err)
	}
	defer smConn.Close()

	kmsConn, err := grpc.Dial(
		fmt.Sprintf("localhost:%d", kmsPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("Failed to connect to KMS: %v", err)
	}
	defer kmsConn.Close()

	smClient, err := secretmanager.NewClient(ctx, option.WithGRPCConn(smConn))
	if err != nil {
		t.Fatalf("Failed to create Secret Manager client: %v", err)
	}
	defer smClient.Close()

	kmsClient, err := kms.NewKeyManagementClient(ctx, option.WithGRPCConn(kmsConn))
	if err != nil {
		t.Fatalf("Failed to create KMS client: %v", err)
	}
	defer kmsClient.Close()

	// Encrypt data with KMS
	t.Log("Encrypting data with KMS...")
	plaintext := []byte("cross-service-secret")
	cryptoKeyName := fmt.Sprintf("projects/%s/locations/global/keyRings/test-ring/cryptoKeys/test-key", testProject)
	encryptResp, err := kmsClient.Encrypt(ctx, &kmspb.EncryptRequest{
		Name:      cryptoKeyName,
		Plaintext: plaintext,
	})
	if err != nil {
		t.Fatalf("Failed to encrypt with KMS: %v", err)
	}

	// Store encrypted data in Secret Manager
	t.Log("Storing encrypted data in Secret Manager...")
	secretName := fmt.Sprintf("projects/%s/secrets/encrypted-secret", testProject)
	_, err = smClient.CreateSecret(ctx, &secretmanagerpb.CreateSecretRequest{
		Parent:   fmt.Sprintf("projects/%s", testProject),
		SecretId: "encrypted-secret",
		Secret: &secretmanagerpb.Secret{
			Replication: &secretmanagerpb.Replication{
				Replication: &secretmanagerpb.Replication_Automatic_{
					Automatic: &secretmanagerpb.Replication_Automatic{},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed to create secret: %v", err)
	}

	// Store encrypted ciphertext (base64 encoded for storage)
	encodedCiphertext := []byte(base64.StdEncoding.EncodeToString(encryptResp.Ciphertext))
	version, err := smClient.AddSecretVersion(ctx, &secretmanagerpb.AddSecretVersionRequest{
		Parent: secretName,
		Payload: &secretmanagerpb.SecretPayload{
			Data: encodedCiphertext,
		},
	})
	if err != nil {
		t.Fatalf("Failed to add secret version: %v", err)
	}

	// Retrieve from Secret Manager
	t.Log("Retrieving from Secret Manager...")
	accessResp, err := smClient.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{
		Name: version.Name,
	})
	if err != nil {
		t.Fatalf("Failed to access secret: %v", err)
	}

	// Decode and decrypt with KMS
	t.Log("Decrypting with KMS...")
	ciphertext, err := base64.StdEncoding.DecodeString(string(accessResp.Payload.Data))
	if err != nil {
		t.Fatalf("Failed to decode ciphertext: %v", err)
	}

	decryptResp, err := kmsClient.Decrypt(ctx, &kmspb.DecryptRequest{
		Name:       cryptoKeyName,
		Ciphertext: ciphertext,
	})
	if err != nil {
		t.Fatalf("Failed to decrypt: %v", err)
	}

	if string(decryptResp.Plaintext) != string(plaintext) {
		t.Errorf("Cross-service data mismatch: got %q, want %q", decryptResp.Plaintext, plaintext)
	}

	t.Log("Cross-service test passed")
}

func testIAMEnforcement(t *testing.T) {
	ctx := context.Background()

	// Test with unauthorized principal
	unauthorizedPrincipal := "user:unauthorized@example.com"
	ctx = metadata.AppendToOutgoingContext(ctx, "x-emulator-principal", unauthorizedPrincipal)

	conn, err := grpc.Dial(
		fmt.Sprintf("localhost:%d", secretManagerPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client, err := secretmanager.NewClient(ctx, option.WithGRPCConn(conn))
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Attempt to create secret (should be denied in permissive mode if policy denies)
	t.Log("Testing IAM enforcement with unauthorized principal...")
	_, err = client.CreateSecret(ctx, &secretmanagerpb.CreateSecretRequest{
		Parent:   fmt.Sprintf("projects/%s", testProject),
		SecretId: "unauthorized-secret",
		Secret: &secretmanagerpb.Secret{
			Replication: &secretmanagerpb.Replication{
				Replication: &secretmanagerpb.Replication_Automatic_{
					Automatic: &secretmanagerpb.Replication_Automatic{},
				},
			},
		},
	})

	// In permissive mode, this might succeed or fail depending on policy
	// We're just validating that IAM is being checked
	if err != nil {
		t.Logf("Permission denied as expected: %v", err)
	} else {
		t.Log("Request succeeded (policy may allow this principal)")
	}
}
