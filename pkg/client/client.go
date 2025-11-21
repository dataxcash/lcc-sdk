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
// The client communicates with LCC server to check feature authorization.
//
// Authorization Model:
// - Old model (deprecated): YAML defines tier requirements, License validates tier match
// - New model (recommended): License directly controls which features are enabled
//
// The client is backward compatible and works with both old and new License formats.
type Client struct {
	baseURL    string
	productID  string
	productVer string

	httpClient *http.Client
	keyPair    *auth.KeyPair
	signer     *auth.RequestSigner
	cache      *featureCache
	instanceID string

	mu sync.RWMutex
}

// FeatureStatus represents the status of a feature check
type FeatureStatus struct {
	Enabled bool   `json:"enabled"`
	Reason  string `json:"reason,omitempty"`

	// Optional quota information (for consumption limits)
	Quota *QuotaInfo `json:"quota_info,omitempty"`

	// Optional demo limits for different control types
	MaxCapacity    int     `json:"max_capacity,omitempty"`
	MaxTPS         float64 `json:"max_tps,omitempty"`
	MaxConcurrency int     `json:"max_concurrency,omitempty"`
}

// QuotaInfo mirrors the server-side SDKQuotaInfo structure
type QuotaInfo struct {
	Limit     int   `json:"limit"`
	Used      int   `json:"used"`
	Remaining int   `json:"remaining"`
	ResetAt   int64 `json:"reset_at"`
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

// concurrencyState tracks in-process concurrency per (instanceID, featureID).
// This is a package-level variable for simplicity in the demo. In a real
// implementation this should be moved to a dedicated structure with proper
// lifecycle management.
var concurrencyState = make(map[string]int)

// NewClient creates a new LCC client using a freshly generated key pair
func NewClient(cfg *config.SDKConfig) (*Client, error) {
	kp, err := auth.GenerateKeyPair()
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %w", err)
	}
	return NewClientWithKeyPair(cfg, kp)
}

// NewClientWithKeyPair creates a client using the provided key pair
func NewClientWithKeyPair(cfg *config.SDKConfig, keyPair *auth.KeyPair) (*Client, error) {
	if keyPair == nil {
		return nil, fmt.Errorf("keyPair is nil")
	}
	instanceID, err := keyPair.GetFingerprint()
	if err != nil {
		return nil, fmt.Errorf("failed to get fingerprint: %w", err)
	}
	client := &Client{
		baseURL:    cfg.LCCURL,
		productID:  cfg.ProductID,
		productVer: cfg.ProductVersion,
		httpClient: &http.Client{ Timeout: cfg.Timeout },
		keyPair:    keyPair,
		signer:     auth.NewRequestSigner(keyPair),
		instanceID: instanceID,
		cache:      &featureCache{ data: make(map[string]*cacheEntry), ttl: cfg.CacheTTL },
	}
	return client, nil
}

// Register registers this application instance with LCC
func (c *Client) Register() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	pubPEM, err := c.keyPair.GetPublicKeyPEM()
	if err != nil {
		return fmt.Errorf("failed to export public key: %w", err)
	}

	reqBody := map[string]interface{}{
		"product_id": c.productID,
		"version":    c.productVer,
		"public_key": pubPEM,
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

// CheckFeature checks if a feature is enabled in the License.
// Authorization is controlled by the License file, not by YAML configuration.
// The YAML config only maps feature IDs to functions (technical mapping).
//
// The License file determines:
// - Whether the feature is enabled
// - Quota limits (if applicable)
// - Capacity, TPS, and concurrency limits
//
// Returns FeatureStatus with:
// - Enabled: true if feature is authorized in license
// - Reason: explanation if disabled (e.g., "feature_not_in_license", "quota_exceeded")
// - Quota: quota information if applicable
// - Capacity/TPS/Concurrency: limits from license
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
		FeatureID      string     `json:"feature_id"`
		Enabled        bool       `json:"enabled"`
		Reason         string     `json:"reason"`
		QuotaInfo      *QuotaInfo `json:"quota_info,omitempty"`
		MaxCapacity    int        `json:"max_capacity,omitempty"`
		MaxTPS         float64    `json:"max_tps,omitempty"`
		MaxConcurrency int        `json:"max_concurrency,omitempty"`
		CacheTTL       int        `json:"cache_ttl"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &FeatureStatus{
		Enabled:        result.Enabled,
		Reason:         result.Reason,
		Quota:          result.QuotaInfo,
		MaxCapacity:    result.MaxCapacity,
		MaxTPS:         result.MaxTPS,
		MaxConcurrency: result.MaxConcurrency,
	}, nil
}

// Consume performs a consumption-style check+usage for an event-based feature.
// Typical use: MAXCALL, license generation, export count, etc.
// It first checks the feature, then reports usage if allowed.
func (c *Client) Consume(featureID string, amount int, meta map[string]any) (bool, int, string, error) {
	status, err := c.CheckFeature(featureID)
	if err != nil {
		return false, 0, "check_error", err
	}
	if !status.Enabled {
		remaining := 0
		if status.Quota != nil {
			remaining = status.Quota.Remaining
		}
		return false, remaining, status.Reason, nil
	}

	// Report usage as a single event (server-side quota tracking)
	if err := c.ReportUsage(featureID, float64(amount)); err != nil {
		return false, 0, "usage_error", err
	}

	remaining := 0
	if status.Quota != nil {
		// Note: this is approximate, real remaining will be updated on next check
		remaining = status.Quota.Remaining - amount
		if remaining < 0 {
			remaining = 0
		}
	}

	return true, remaining, "ok", nil
}

// CheckCapacity compares an APP-provided currentUsed against the license-defined
// MaxCapacity for the given feature. SDK does not compute current usage itself.
func (c *Client) CheckCapacity(featureID string, currentUsed int) (bool, int, string, error) {
	status, err := c.CheckFeature(featureID)
	if err != nil {
		return false, 0, "check_error", err
	}

	max := status.MaxCapacity
	if max <= 0 {
		// No capacity configured for this feature
		return false, 0, "no_capacity_limit", nil
	}

	if currentUsed > max {
		return false, max, "capacity_exceeded", nil
	}

	return true, max, "ok", nil
}

// CheckTPS compares an APP-provided currentTPS against the license-defined
// MaxTPS for the given feature.
func (c *Client) CheckTPS(featureID string, currentTPS float64) (bool, float64, string, error) {
	status, err := c.CheckFeature(featureID)
	if err != nil {
		return false, 0, "check_error", err
	}

	max := status.MaxTPS
	if max <= 0 {
		return false, 0, "no_tps_limit", nil
	}

	if currentTPS > max {
		return false, max, "tps_exceeded", nil
	}

	return true, max, "ok", nil
}

// AcquireSlot implements a simple in-process concurrency control based on
// MaxConcurrency from the feature check. It returns a release function that
// must be called to free the slot.
func (c *Client) AcquireSlot(featureID string, meta map[string]any) (func(), bool, string, error) {
	status, err := c.CheckFeature(featureID)
	if err != nil {
		return func() {}, false, "check_error", err
	}

	max := status.MaxConcurrency
	if max <= 0 {
		return func() {}, false, "no_concurrency_limit", nil
	}

	// Simple per-feature counter; no cross-process coordination.
	// For demo purposes only.
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cache == nil {
		c.cache = &featureCache{data: make(map[string]*cacheEntry), ttl: 0}
	}

	// Reuse cache map to store a simple counter via cacheEntry.Total field is not ideal,
	// but to keep changes minimal, we track concurrency in a dedicated map.
	// For clarity, we keep a separate field on Client.

	// Lazy init per-feature concurrency map
	if cConcurrency, ok := concurrencyState[c.instanceID]; ok {
		_ = cConcurrency
	}

	// Global in-process map: instanceID+featureID -> current count
	key := c.instanceID + "::" + featureID
	current := concurrencyState[key]
	if current >= max {
		return func() {}, false, "concurrency_exceeded", nil
	}

	concurrencyState[key] = current + 1

	release := func() {
		c.mu.Lock()
		defer c.mu.Unlock()
		cur := concurrencyState[key]
		if cur <= 1 {
			delete(concurrencyState, key)
		} else {
			concurrencyState[key] = cur - 1
		}
	}

	return release, true, "ok", nil
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
