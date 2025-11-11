package auth

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
)

// KeyPair represents an RSA key pair for self-signed authentication
type KeyPair struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
}

// GenerateKeyPair generates a new RSA key pair
// Key size is 2048 bits as per specification
func GenerateKeyPair() (*KeyPair, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate RSA key: %w", err)
	}

	return &KeyPair{
		privateKey: privateKey,
		publicKey:  &privateKey.PublicKey,
	}, nil
}

// Sign signs data using the private key with PKCS#1 v1.5 padding
func (kp *KeyPair) Sign(data []byte) ([]byte, error) {
	if kp.privateKey == nil {
		return nil, fmt.Errorf("private key is nil")
	}

	// Hash the data with SHA-256
	hashed := sha256.Sum256(data)

	// Sign with RSA PKCS#1 v1.5
	signature, err := rsa.SignPKCS1v15(rand.Reader, kp.privateKey, crypto.SHA256, hashed[:])
	if err != nil {
		return nil, fmt.Errorf("failed to sign data: %w", err)
	}

	return signature, nil
}

// Verify verifies a signature using the public key
func (kp *KeyPair) Verify(data []byte, signature []byte) error {
	if kp.publicKey == nil {
		return fmt.Errorf("public key is nil")
	}

	// Hash the data
	hashed := sha256.Sum256(data)

	// Verify signature
	err := rsa.VerifyPKCS1v15(kp.publicKey, crypto.SHA256, hashed[:], signature)
	if err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}

	return nil
}

// GetPublicKeyPEM exports the public key in PEM format
func (kp *KeyPair) GetPublicKeyPEM() (string, error) {
	if kp.publicKey == nil {
		return "", fmt.Errorf("public key is nil")
	}

	// Marshal public key to PKIX format
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(kp.publicKey)
	if err != nil {
		return "", fmt.Errorf("failed to marshal public key: %w", err)
	}

	// Create PEM block
	pemBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubKeyBytes,
	}

	// Encode to PEM
	pemBytes := pem.EncodeToMemory(pemBlock)
	return string(pemBytes), nil
}

// GetPublicKeyDER exports the public key in DER format
func (kp *KeyPair) GetPublicKeyDER() ([]byte, error) {
	if kp.publicKey == nil {
		return nil, fmt.Errorf("public key is nil")
	}

	return x509.MarshalPKIXPublicKey(kp.publicKey)
}

// GetFingerprint returns the SHA-256 fingerprint of the public key
// This can be used as a unique identifier for the instance
func (kp *KeyPair) GetFingerprint() (string, error) {
	pubKeyDER, err := kp.GetPublicKeyDER()
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(pubKeyDER)
	return hex.EncodeToString(hash[:]), nil
}

// ParsePublicKeyFromPEM parses a public key from PEM format
func ParsePublicKeyFromPEM(pemData []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	if block.Type != "PUBLIC KEY" {
		return nil, fmt.Errorf("invalid PEM type: %s", block.Type)
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("not an RSA public key")
	}

	return rsaPub, nil
}

// VerifySignatureWithPublicKey verifies a signature using a public key in PEM format
func VerifySignatureWithPublicKey(publicKeyPEM []byte, data []byte, signature []byte) error {
	publicKey, err := ParsePublicKeyFromPEM(publicKeyPEM)
	if err != nil {
		return err
	}

	hashed := sha256.Sum256(data)
	err = rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, hashed[:], signature)
	if err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}

	return nil
}

// Destroy securely wipes the private key from memory
func (kp *KeyPair) Destroy() {
	if kp.privateKey != nil {
		// Zero out the private key components
		// Note: This provides basic cleanup, but Go's GC may have made copies
		if kp.privateKey.D != nil {
			kp.privateKey.D.SetInt64(0)
		}
		if kp.privateKey.Primes != nil {
			for _, prime := range kp.privateKey.Primes {
				if prime != nil {
					prime.SetInt64(0)
				}
			}
		}
		kp.privateKey = nil
	}
	kp.publicKey = nil
}
