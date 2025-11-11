package auth

import (
	"bytes"
	"net/http/httptest"
	"testing"
)

func TestGenerateKeyPair(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	if kp.privateKey == nil {
		t.Error("private key is nil")
	}

	if kp.publicKey == nil {
		t.Error("public key is nil")
	}
}

func TestKeyPair_SignAndVerify(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	data := []byte("test data to sign")

	// Sign
	signature, err := kp.Sign(data)
	if err != nil {
		t.Fatalf("Sign() error = %v", err)
	}

	// Verify
	err = kp.Verify(data, signature)
	if err != nil {
		t.Errorf("Verify() error = %v, want nil", err)
	}

	// Verify with wrong data should fail
	wrongData := []byte("wrong data")
	err = kp.Verify(wrongData, signature)
	if err == nil {
		t.Error("Verify() with wrong data should fail")
	}
}

func TestKeyPair_GetPublicKeyPEM(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	pem, err := kp.GetPublicKeyPEM()
	if err != nil {
		t.Fatalf("GetPublicKeyPEM() error = %v", err)
	}

	if pem == "" {
		t.Error("PEM is empty")
	}

	// Should contain PEM header
	if !bytes.Contains([]byte(pem), []byte("BEGIN PUBLIC KEY")) {
		t.Error("PEM does not contain header")
	}
}

func TestKeyPair_GetFingerprint(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	fp1, err := kp.GetFingerprint()
	if err != nil {
		t.Fatalf("GetFingerprint() error = %v", err)
	}

	fp2, err := kp.GetFingerprint()
	if err != nil {
		t.Fatalf("GetFingerprint() error = %v", err)
	}

	// Fingerprint should be consistent
	if fp1 != fp2 {
		t.Errorf("fingerprints differ: %s != %s", fp1, fp2)
	}

	// Fingerprint should be hex-encoded SHA-256 (64 chars)
	if len(fp1) != 64 {
		t.Errorf("fingerprint length = %d, want 64", len(fp1))
	}
}

func TestParsePublicKeyFromPEM(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	pemStr, err := kp.GetPublicKeyPEM()
	if err != nil {
		t.Fatalf("GetPublicKeyPEM() error = %v", err)
	}

	// Parse it back
	parsedKey, err := ParsePublicKeyFromPEM([]byte(pemStr))
	if err != nil {
		t.Fatalf("ParsePublicKeyFromPEM() error = %v", err)
	}

	// Should be able to verify signature with parsed key
	data := []byte("test data")
	signature, _ := kp.Sign(data)

	// Create temporary keypair with parsed public key for verification
	tempKP := &KeyPair{publicKey: parsedKey}
	err = tempKP.Verify(data, signature)
	if err != nil {
		t.Errorf("Verify() with parsed key error = %v", err)
	}
}

func TestRequestSigner_SignRequest(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	signer := NewRequestSigner(kp)

	// Create test request
	body := []byte(`{"test": "data"}`)
	req := httptest.NewRequest("POST", "/api/v1/test", bytes.NewReader(body))

	// Sign request
	err = signer.SignRequest(req)
	if err != nil {
		t.Fatalf("SignRequest() error = %v", err)
	}

	// Check headers are set
	headers := []string{
		"X-LCC-PublicKey",
		"X-LCC-Timestamp",
		"X-LCC-Nonce",
		"X-LCC-Signature",
	}

	for _, header := range headers {
		if req.Header.Get(header) == "" {
			t.Errorf("Header %s not set", header)
		}
	}
}

func TestRequestSigner_SignAndVerify(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	signer := NewRequestSigner(kp)

	// Create and sign request
	body := []byte(`{"feature": "test"}`)
	req := httptest.NewRequest("POST", "/api/v1/features/check", bytes.NewReader(body))

	err = signer.SignRequest(req)
	if err != nil {
		t.Fatalf("SignRequest() error = %v", err)
	}

	// Verify request
	err = VerifyRequest(req)
	if err != nil {
		t.Errorf("VerifyRequest() error = %v, want nil", err)
	}
}

func TestVerifyRequest_InvalidSignature(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	signer := NewRequestSigner(kp)

	// Create and sign request
	body := []byte(`{"test": "data"}`)
	req := httptest.NewRequest("POST", "/api/v1/test", bytes.NewReader(body))

	err = signer.SignRequest(req)
	if err != nil {
		t.Fatalf("SignRequest() error = %v", err)
	}

	// Tamper with signature
	req.Header.Set("X-LCC-Signature", "invalid_signature_hex")

	// Verification should fail
	err = VerifyRequest(req)
	if err == nil {
		t.Error("VerifyRequest() should fail with invalid signature")
	}
}

func TestVerifyRequest_MissingHeaders(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/test", nil)

	err := VerifyRequest(req)
	if err == nil {
		t.Error("VerifyRequest() should fail with missing headers")
	}
}

func TestComputeBodyHash(t *testing.T) {
	body := []byte(`{"key": "value"}`)
	hash := ComputeBodyHash(body)

	// Should be hex-encoded SHA-256 (64 chars)
	if len(hash) != 64 {
		t.Errorf("hash length = %d, want 64", len(hash))
	}

	// Same body should produce same hash
	hash2 := ComputeBodyHash(body)
	if hash != hash2 {
		t.Error("hashes differ for same body")
	}

	// Different body should produce different hash
	differentBody := []byte(`{"key": "different"}`)
	hash3 := ComputeBodyHash(differentBody)
	if hash == hash3 {
		t.Error("hashes are same for different bodies")
	}
}

func TestKeyPair_Destroy(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	// Before destroy
	if kp.privateKey == nil {
		t.Fatal("private key should not be nil before destroy")
	}

	// Destroy
	kp.Destroy()

	// After destroy
	if kp.privateKey != nil {
		t.Error("private key should be nil after destroy")
	}
	if kp.publicKey != nil {
		t.Error("public key should be nil after destroy")
	}
}

func BenchmarkGenerateKeyPair(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := GenerateKeyPair()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSign(b *testing.B) {
	kp, _ := GenerateKeyPair()
	data := []byte("benchmark data to sign")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := kp.Sign(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkVerify(b *testing.B) {
	kp, _ := GenerateKeyPair()
	data := []byte("benchmark data to verify")
	signature, _ := kp.Sign(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := kp.Verify(data, signature)
		if err != nil {
			b.Fatal(err)
		}
	}
}
