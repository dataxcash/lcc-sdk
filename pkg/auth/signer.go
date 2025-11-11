package auth

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
)

// RequestSigner signs HTTP requests with RSA signatures
type RequestSigner struct {
	keyPair *KeyPair
}

// NewRequestSigner creates a new request signer with the given key pair
func NewRequestSigner(keyPair *KeyPair) *RequestSigner {
	return &RequestSigner{
		keyPair: keyPair,
	}
}

// SignRequest signs an HTTP request and adds authentication headers
// Headers added:
//   - X-LCC-PublicKey: Base64-encoded public key in PEM format
//   - X-LCC-Timestamp: Unix timestamp in seconds
//   - X-LCC-Nonce: Unique nonce (UUID)
//   - X-LCC-Signature: Hex-encoded signature
func (s *RequestSigner) SignRequest(req *http.Request) error {
	// Generate timestamp and nonce
	timestamp := time.Now().Unix()
	nonce := uuid.New().String()

	// Read and hash request body
	var bodyHash string
	if req.Body != nil {
		bodyBytes, err := io.ReadAll(req.Body)
		if err != nil {
			return fmt.Errorf("failed to read request body: %w", err)
		}

		// Compute SHA-256 of body
		hash := sha256.Sum256(bodyBytes)
		bodyHash = hex.EncodeToString(hash[:])

		// Restore body for actual request
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		req.ContentLength = int64(len(bodyBytes))
	} else {
		// Empty body hash
		emptyHash := sha256.Sum256([]byte{})
		bodyHash = hex.EncodeToString(emptyHash[:])
	}

	// Build canonical string
	// Format: METHOD\nPATH\nBODY_SHA256\nTIMESTAMP\nNONCE
	canonical := fmt.Sprintf("%s\n%s\n%s\n%d\n%s",
		req.Method,
		req.URL.Path,
		bodyHash,
		timestamp,
		nonce,
	)

	// Sign canonical string
	signature, err := s.keyPair.Sign([]byte(canonical))
	if err != nil {
		return fmt.Errorf("failed to sign request: %w", err)
	}

	// Get public key in PEM format
	publicKeyPEM, err := s.keyPair.GetPublicKeyPEM()
	if err != nil {
		return fmt.Errorf("failed to get public key: %w", err)
	}

	// Encode public key as base64
	publicKeyBase64 := base64.StdEncoding.EncodeToString([]byte(publicKeyPEM))

	// Add authentication headers
	req.Header.Set("X-LCC-PublicKey", publicKeyBase64)
	req.Header.Set("X-LCC-Timestamp", strconv.FormatInt(timestamp, 10))
	req.Header.Set("X-LCC-Nonce", nonce)
	req.Header.Set("X-LCC-Signature", hex.EncodeToString(signature))
	req.Header.Set("Content-Type", "application/json")

	return nil
}

// VerifyRequest verifies the signature of an HTTP request
// This is used server-side to verify client requests
func VerifyRequest(req *http.Request) error {
	// Extract headers
	publicKeyBase64 := req.Header.Get("X-LCC-PublicKey")
	timestampStr := req.Header.Get("X-LCC-Timestamp")
	nonce := req.Header.Get("X-LCC-Nonce")
	signatureHex := req.Header.Get("X-LCC-Signature")

	if publicKeyBase64 == "" || timestampStr == "" || nonce == "" || signatureHex == "" {
		return fmt.Errorf("missing authentication headers")
	}

	// Parse timestamp
	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid timestamp: %w", err)
	}

	// Verify timestamp is recent (within 5 minutes)
	now := time.Now().Unix()
	if now-timestamp > 300 || timestamp-now > 60 {
		return fmt.Errorf("timestamp out of range (diff: %d seconds)", now-timestamp)
	}

	// Read and hash request body
	var bodyHash string
	if req.Body != nil {
		bodyBytes, err := io.ReadAll(req.Body)
		if err != nil {
			return fmt.Errorf("failed to read request body: %w", err)
		}

		hash := sha256.Sum256(bodyBytes)
		bodyHash = hex.EncodeToString(hash[:])

		// Restore body
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	} else {
		emptyHash := sha256.Sum256([]byte{})
		bodyHash = hex.EncodeToString(emptyHash[:])
	}

	// Rebuild canonical string
	canonical := fmt.Sprintf("%s\n%s\n%s\n%s\n%s",
		req.Method,
		req.URL.Path,
		bodyHash,
		timestampStr,
		nonce,
	)

	// Decode public key
	publicKeyPEM, err := base64.StdEncoding.DecodeString(publicKeyBase64)
	if err != nil {
		return fmt.Errorf("failed to decode public key: %w", err)
	}

	// Decode signature
	signature, err := hex.DecodeString(signatureHex)
	if err != nil {
		return fmt.Errorf("failed to decode signature: %w", err)
	}

	// Verify signature
	if err := VerifySignatureWithPublicKey(publicKeyPEM, []byte(canonical), signature); err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}

	return nil
}

// BuildCanonicalString builds the canonical string for signing
// This is exposed for testing purposes
func BuildCanonicalString(method, path, bodyHash string, timestamp int64, nonce string) string {
	return fmt.Sprintf("%s\n%s\n%s\n%d\n%s",
		method,
		path,
		bodyHash,
		timestamp,
		nonce,
	)
}

// ComputeBodyHash computes SHA-256 hash of request body
func ComputeBodyHash(body []byte) string {
	hash := sha256.Sum256(body)
	return hex.EncodeToString(hash[:])
}
