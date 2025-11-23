package client

import (
	"bytes"
	"context"
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

	// Heartbeat management
	heartbeatInterval time.Duration
	heartbeatCancel   context.CancelFunc
	heartbeatRunning  bool

	// Zero-intrusion API fields
	helpers    *HelperFunctions
	tpsTracker *tpsTracker

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

const defaultHeartbeatInterval = 5 * time.Second

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

		httpClient: &http.Client{Timeout: cfg.Timeout},
		keyPair:   keyPair,
		signer:    auth.NewRequestSigner(keyPair),
		cache:     &featureCache{data: make(map[string]*cacheEntry), ttl: cfg.CacheTTL},
		instanceID:          instanceID,
		heartbeatInterval:   defaultHeartbeatInterval,
		tpsTracker:          newTPSTracker(),
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

	// Start background heartbeat loop after successful registration
	c.startHeartbeatLoop()

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

// RegisterHelpers registers helper functions for zero-intrusion API usage.
// This enables product-level limit checking without requiring featureID parameters.
//
// Required helpers:
//   - CapacityCounter: MUST be provided if using capacity limits
//
// Optional helpers (SDK provides defaults if not specified):
//   - QuotaConsumer: defaults to consuming 1 unit per call
//   - TPSProvider: defaults to SDK internal TPS tracking
//
// Example:
//   helpers := &client.HelperFunctions{
//       QuotaConsumer: func(ctx context.Context, args ...interface{}) int {
//           return calculateBatchSize(args)
//       },
//       CapacityCounter: func() int {
//           return database.CountActiveUsers()
//       },
//   }
//   client.RegisterHelpers(helpers)
func (c *Client) RegisterHelpers(helpers *HelperFunctions) error {
	if helpers == nil {
		return fmt.Errorf("helpers cannot be nil")
	}

	// Validate required helpers
	if err := helpers.Validate(); err != nil {
		return fmt.Errorf("helper validation failed: %w", err)
	}

	// Set defaults for optional helpers
	helpers.SetDefaults(c)

	c.mu.Lock()
	c.helpers = helpers
	c.mu.Unlock()

	return nil
}

// getInternalTPS returns the current TPS from the internal tracker
func (c *Client) getInternalTPS() float64 {
	if c.tpsTracker == nil {
		return 0
	}
	return c.tpsTracker.getCurrentRate()
}

// checkProductLimits checks product-level limits (not feature-specific)
// This is used by the zero-intrusion API methods
func (c *Client) checkProductLimits() (*FeatureStatus, error) {
	// Use a special product-level feature ID
	// The server should recognize this and return product-level limits
	return c.CheckFeature("__product__")
}

// reportProductUsage reports usage at the product level
func (c *Client) reportProductUsage(amount int) error {
	return c.ReportUsage("__product__", float64(amount))
}

// startHeartbeatLoop starts a background goroutine that periodically
// sends heartbeat requests to LCC. It is idempotent and safe to call
// multiple times; only a single loop will run.
func (c *Client) startHeartbeatLoop() {
	c.mu.Lock()
	if c.heartbeatRunning {
		c.mu.Unlock()
		return
	}
	interval := c.heartbeatInterval
	if interval <= 0 {
		interval = defaultHeartbeatInterval
	}
	ctx, cancel := context.WithCancel(context.Background())
	c.heartbeatCancel = cancel
	c.heartbeatRunning = true
	c.mu.Unlock()

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_ = c.sendHeartbeat()
			}
		}
	}()
}

// sendHeartbeat sends a single heartbeat request to LCC.
// Errors are returned to the caller but are not retried here.
func (c *Client) sendHeartbeat() error {
	payload := map[string]interface{}{
		"version": c.productVer,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal heartbeat payload: %w", err)
	}

	req, err := http.NewRequest("POST", c.baseURL+"/api/v1/sdk/heartbeat", bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create heartbeat request: %w", err)
	}

	if err := c.signer.SignRequest(req); err != nil {
		return fmt.Errorf("failed to sign heartbeat request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("heartbeat request failed: %w", err)
	}
	defer resp.Body.Close()

	// Drain response body and ignore content; heartbeat is best-effort
	_, _ = io.Copy(io.Discard, resp.Body)

	return nil
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

// ========== Zero-Intrusion Product-Level API (New) ==========

// ReleaseFunc is a function that releases a concurrency slot
type ReleaseFunc func()

// Consume consumes quota from the product-level quota pool.
// This is the zero-intrusion API that does not require featureID.
//
// The amount parameter specifies how many quota units to consume.
// For simple use cases where each call consumes 1 unit, pass 1.
//
// Returns:
//   - allowed: true if quota is available
//   - remaining: remaining quota after consumption
//   - error: any error during the check
//
// Example:
//   allowed, remaining, err := client.Consume(1)
//   if err != nil || !allowed {
//       return fmt.Errorf("quota exceeded")
//   }
func (c *Client) Consume(amount int) (bool, int, error) {
	// Record TPS for internal tracking
	if c.tpsTracker != nil {
		c.tpsTracker.RecordRequest()
	}

	// Check product-level quota
	status, err := c.checkProductLimits()
	if err != nil {
		return false, 0, err
	}

	if !status.Enabled {
		remaining := 0
		if status.Quota != nil {
			remaining = status.Quota.Remaining
		}
		return false, remaining, fmt.Errorf("quota exceeded: %s", status.Reason)
	}

	// Report usage
	if err := c.reportProductUsage(amount); err != nil {
		return false, 0, err
	}

	remaining := 0
	if status.Quota != nil {
		remaining = status.Quota.Remaining - amount
		if remaining < 0 {
			remaining = 0
		}
	}

	return true, remaining, nil
}

// ConsumeWithContext uses the registered QuotaConsumer helper function
// to calculate consumption amount based on function arguments.
//
// This is designed for code generator integration where the wrapper
// function passes the original function arguments to calculate dynamic
// consumption amounts.
//
// Returns:
//   - allowed: true if quota is available
//   - remaining: remaining quota after consumption
//   - error: any error during the check
//
// Example:
//   allowed, remaining, err := client.ConsumeWithContext(ctx, batchSize, userID)
//   if err != nil || !allowed {
//       return fmt.Errorf("quota exceeded")
//   }
func (c *Client) ConsumeWithContext(ctx context.Context, args ...interface{}) (bool, int, error) {
	c.mu.RLock()
	helpers := c.helpers
	c.mu.RUnlock()

	if helpers == nil || helpers.QuotaConsumer == nil {
		return false, 0, fmt.Errorf("QuotaConsumer helper not registered")
	}

	amount := helpers.QuotaConsumer(ctx, args...)
	return c.Consume(amount)
}

// ConsumeDeprecated performs a consumption-style check+usage for an event-based feature.
// Typical use: MAXCALL, license generation, export count, etc.
// It first checks the feature, then reports usage if allowed.
//
// DEPRECATED: Use product-level Consume() or ConsumeWithContext() instead.
// This method is kept for backward compatibility only.
func (c *Client) ConsumeDeprecated(featureID string, amount int, meta map[string]any) (bool, int, string, error) {
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

// CheckCapacity checks current usage against product-level capacity limit.
// This is the zero-intrusion API that does not require featureID.
//
// The currentUsed parameter should be the current number of resources in use
// (e.g., active users, open connections, created items).
//
// Returns:
//   - allowed: true if capacity is available
//   - maxCapacity: the maximum capacity limit
//   - error: any error during the check
//
// Example:
//   currentUsers := database.CountActiveUsers()
//   allowed, max, err := client.CheckCapacity(currentUsers)
//   if err != nil || !allowed {
//       return fmt.Errorf("capacity exceeded: %d/%d", currentUsers, max)
//   }
func (c *Client) CheckCapacity(currentUsed int) (bool, int, error) {
	status, err := c.checkProductLimits()
	if err != nil {
		return false, 0, err
	}

	maxCapacity := status.MaxCapacity
	if maxCapacity <= 0 {
		return false, 0, fmt.Errorf("no capacity limit configured")
	}

	if currentUsed >= maxCapacity {
		return false, maxCapacity, fmt.Errorf("capacity exceeded: %d >= %d", currentUsed, maxCapacity)
	}

	return true, maxCapacity, nil
}

// CheckCapacityWithHelper uses the registered CapacityCounter helper function
// to automatically get the current usage count.
//
// This requires that a CapacityCounter helper has been registered via RegisterHelpers().
//
// Returns:
//   - allowed: true if capacity is available
//   - maxCapacity: the maximum capacity limit
//   - error: any error during the check
//
// Example:
//   allowed, max, err := client.CheckCapacityWithHelper()
//   if err != nil || !allowed {
//       return fmt.Errorf("capacity exceeded")
//   }
func (c *Client) CheckCapacityWithHelper() (bool, int, error) {
	c.mu.RLock()
	helpers := c.helpers
	c.mu.RUnlock()

	if helpers == nil || helpers.CapacityCounter == nil {
		return false, 0, fmt.Errorf("CapacityCounter helper not registered (required)")
	}

	currentUsed := helpers.CapacityCounter()
	return c.CheckCapacity(currentUsed)
}

// CheckCapacityDeprecated compares an APP-provided currentUsed against the license-defined
// MaxCapacity for the given feature. SDK does not compute current usage itself.
//
// DEPRECATED: Use product-level CheckCapacity() or CheckCapacityWithHelper() instead.
// This method is kept for backward compatibility only.
func (c *Client) CheckCapacityDeprecated(featureID string, currentUsed int) (bool, int, string, error) {
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

// CheckTPS checks current TPS against product-level limit.
// This is the zero-intrusion API that does not require featureID.
//
// Uses the registered TPSProvider helper if available, otherwise uses
// SDK internal TPS tracking.
//
// Returns:
//   - allowed: true if TPS is within limit
//   - maxTPS: the maximum TPS limit
//   - error: any error during the check
//
// Example:
//   allowed, maxTPS, err := client.CheckTPS()
//   if err != nil || !allowed {
//       return fmt.Errorf("TPS exceeded: max=%.2f", maxTPS)
//   }
func (c *Client) CheckTPS() (bool, float64, error) {
	// Get current TPS from helper or internal tracker
	currentTPS := c.getCurrentTPS()

	// Check against product limit
	status, err := c.checkProductLimits()
	if err != nil {
		return false, 0, err
	}

	maxTPS := status.MaxTPS
	if maxTPS <= 0 {
		return true, 0, nil // No TPS limit configured
	}

	if currentTPS > maxTPS {
		return false, maxTPS, fmt.Errorf("TPS exceeded: %.2f > %.2f", currentTPS, maxTPS)
	}

	return true, maxTPS, nil
}

// getCurrentTPS gets TPS from helper or internal tracker
func (c *Client) getCurrentTPS() float64 {
	c.mu.RLock()
	helpers := c.helpers
	c.mu.RUnlock()

	if helpers != nil && helpers.TPSProvider != nil {
		return helpers.TPSProvider()
	}
	return c.getInternalTPS()
}

// CheckTPSDeprecated compares an APP-provided currentTPS against the license-defined
// MaxTPS for the given feature.
//
// DEPRECATED: Use product-level CheckTPS() instead.
// This method is kept for backward compatibility only.
func (c *Client) CheckTPSDeprecated(featureID string, currentTPS float64) (bool, float64, string, error) {
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

// AcquireSlot acquires a slot from the product-level concurrency pool.
// This is the zero-intrusion API that does not require featureID.
//
// Returns a release function that MUST be called when the operation completes
// to free the concurrency slot. Use defer to ensure proper cleanup.
//
// Returns:
//   - release: function to release the slot (MUST be called)
//   - allowed: true if slot was acquired
//   - error: any error during the check
//
// Example:
//   release, allowed, err := client.AcquireSlot()
//   if err != nil || !allowed {
//       return fmt.Errorf("concurrency limit exceeded")
//   }
//   defer release()
//   // ... perform operation ...
func (c *Client) AcquireSlot() (ReleaseFunc, bool, error) {
	status, err := c.checkProductLimits()
	if err != nil {
		return func() {}, false, err
	}

	maxConcurrency := status.MaxConcurrency
	if maxConcurrency <= 0 {
		return func() {}, false, fmt.Errorf("no concurrency limit configured")
	}

	// Acquire from product-level pool
	c.mu.Lock()
	defer c.mu.Unlock()

	key := c.instanceID + "::__product__"
	current := concurrencyState[key]

	if current >= maxConcurrency {
		return func() {}, false, fmt.Errorf("concurrency exceeded: %d >= %d", current, maxConcurrency)
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

	return release, true, nil
}

// AcquireSlotDeprecated implements a simple in-process concurrency control based on
// MaxConcurrency from the feature check. It returns a release function that
// must be called to free the slot.
//
// DEPRECATED: Use product-level AcquireSlot() instead.
// This method is kept for backward compatibility only.
func (c *Client) AcquireSlotDeprecated(featureID string, meta map[string]any) (func(), bool, string, error) {
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

	// Stop heartbeat loop if running
	if c.heartbeatCancel != nil {
		c.heartbeatCancel()
		c.heartbeatCancel = nil
		c.heartbeatRunning = false
	}

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
