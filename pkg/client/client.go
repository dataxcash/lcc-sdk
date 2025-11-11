package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/yourorg/lcc-sdk/pkg/auth"
	"github.com/yourorg/lcc-sdk/pkg/config"
)

// Client represents an LCC client instance
type Client struct {
	baseURL       string
	productID     string
	productVer    string
	httpClient    *http.Client
	keyPair       *auth.KeyPair
	signer        *auth.RequestSigner
	cache         *featureCache
	instanceID    string
	mu            sync.RWMutex
}

// FeatureStatus represents the status of a feature check
type FeatureStatus struct {
	Enabled   bool    `json:"enabled"`
	Tier      string  `json:"tier"`
	Reason    string  `json:"reason,omitempty"`
	ExpiresAt int64   `json:"expiresAt,omitempty"`
	Remaining float64 `json:"remaining,omitempty"`
	Total     float64 `json:"total,omitempty"`
}

// featureCache caches feature check results
type featureCache struct {
	data map[string]*cacheEntry
	ttl  time.Duration
	mu   sync.RWMutex
}

type cacheEntry struct {
	status    *FeatureStatus
	expiresAt time.Time
}

// NewClient creates a new LCC client
func NewClient(cfg *config.SDKConfig) (*Client, error) {
	// Generate key pair
	keyPair, err := auth.GenerateKeyPair()
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %w", err)
	}

	// Get instance ID (fingerprint of public key)
	instanceID, err := keyPair.GetFingerprint()
	if err != nil {
		return nil, fmt.Errorf("failed to get fingerprint: %w", err)
	}

	client := &Client{
		baseURL:    cfg.LCCURL,
		productID:  cfg.ProductID,
		productVer: cfg.ProductVersion,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		keyPair:    keyPair,
		signer:     auth.NewRequestSigner(keyPair),
		instanceID: instanceID,
		cache: &featureCache{
			data: make(map[string]*cacheEntry),
			ttl:  cfg.CacheTTL,
		},
	}

	return client, nil
}

// Register registers this application instance with LCC
func (c *Client) Register() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	reqBody := map[string]interface{}{
		"product_id": c.productID,
		"version":    c.productVer,
		"public_key": c.keyPair.GetPublicKeyPEM(),
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", c.baseURL+"/api/v1/sdk/register", bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Sign request
	if err := c.signer.SignRequest(req); err != nil {
		return fmt.Errorf("failed to sign request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("registration failed: status=%d, body=%s", resp.StatusCode, string(body))
	}

	return nil
}

// CheckFeature checks if a feature is enabled
func (c *Client) CheckFeature(featureID string) (*FeatureStatus, error) {
	// Check cache first
	if status := c.cache.get(featureID); status != nil {
		return status, nil
	}

	// Query LCC
	status, err := c.queryFeature(featureID)
	if err != nil {
		return nil, err
	}

	// Cache result
	c.cache.set(featureID, status)

	return status, nil
}

// queryFeature queries LCC for feature status
func (c *Client) queryFeature(featureID string) (*FeatureStatus, error) {
	url := fmt.Sprintf("%s/api/v1/sdk/features/%s/check", c.baseURL, featureID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Sign request
	if err := c.signer.SignRequest(req); err != nil {
		return nil, fmt.Errorf("failed to sign request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("feature check failed: status=%d, body=%s", resp.StatusCode, string(body))
	}

	var result struct {
		FeatureID string `json:"feature_id"`
		Enabled   bool   `json:"enabled"`
		Reason    string `json:"reason"`
		CacheTTL  int    `json:"cache_ttl"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &FeatureStatus{
		Enabled: result.Enabled,
		Reason:  result.Reason,
	}, nil
}

// ReportUsage reports feature usage to LCC
func (c *Client) ReportUsage(featureID string, amount float64) error {
	reqBody := map[string]interface{}{
		"instance_id": c.instanceID,
		"feature_id":  featureID,
		"count":       int(amount),
		"timestamp":   time.Now().Unix(),
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", c.baseURL+"/api/v1/sdk/usage", bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Sign request
	if err := c.signer.SignRequest(req); err != nil {
		return fmt.Errorf("failed to sign request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("usage report failed: status=%d, body=%s", resp.StatusCode, string(body))
	}

	return nil
}

// GetInstanceID returns the instance ID (public key fingerprint)
func (c *Client) GetInstanceID() string {
	return c.instanceID
}

// Close cleans up the client resources
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.keyPair != nil {
		c.keyPair.Destroy()
		c.keyPair = nil
	}

	return nil
}

// Cache methods

func (fc *featureCache) get(featureID string) *FeatureStatus {
	fc.mu.RLock()
	defer fc.mu.RUnlock()

	entry, exists := fc.data[featureID]
	if !exists {
		return nil
	}

	// Check if expired
	if time.Now().After(entry.expiresAt) {
		return nil
	}

	return entry.status
}

func (fc *featureCache) set(featureID string, status *FeatureStatus) {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	fc.data[featureID] = &cacheEntry{
		status:    status,
		expiresAt: time.Now().Add(fc.ttl),
	}
}

func (fc *featureCache) clear() {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	fc.data = make(map[string]*cacheEntry)
}

// ClearCache clears the feature cache
func (c *Client) ClearCache() {
	c.cache.clear()
}
