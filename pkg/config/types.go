package config

import "time"

// Manifest represents the complete lcc-features.yaml configuration
type Manifest struct {
	SDK      SDKConfig       `yaml:"sdk"`
	Features []FeatureConfig `yaml:"features"`
}

// SDKConfig contains global SDK configuration
type SDKConfig struct {
	LCCURL         string        `yaml:"lcc_url"`
	ProductID      string        `yaml:"product_id"`
	ProductVersion string        `yaml:"product_version"`
	CheckInterval  time.Duration `yaml:"check_interval"`
	CacheTTL       time.Duration `yaml:"cache_ttl"`
	FailOpen       bool          `yaml:"fail_open"`
	Timeout        time.Duration `yaml:"timeout"`
	MaxRetries     int           `yaml:"max_retries"`

	// Product-level limits (Zero-Intrusion API)
	// These limits apply to the entire product, not individual features
	Limits *ProductLimits `yaml:"limits,omitempty"`
}

// FeatureConfig defines a single protected feature
// This structure maps feature IDs to functions (technical mapping)
// Authorization control (enabled/disabled, quotas) is defined in the License file
type FeatureConfig struct {
	ID          string          `yaml:"id"`
	Name        string          `yaml:"name"`
	Description string          `yaml:"description"`
	
	// Deprecated: Tier is no longer used for authorization checks.
	// License file now controls feature enablement directly.
	// This field is kept for backward compatibility only.
	Tier        string          `yaml:"tier,omitempty"`
	
	Required    bool            `yaml:"required"`
	Intercept   InterceptConfig `yaml:"intercept"`
	Fallback    *InterceptConfig `yaml:"fallback,omitempty"`
	
	// Deprecated: Quota is no longer defined in YAML.
	// Quota limits should be defined in the License file.
	// This field is kept for backward compatibility only.
	Quota       *QuotaConfig    `yaml:"quota,omitempty"`
	
	Condition   *ConditionConfig `yaml:"condition,omitempty"`
	OnDeny      *OnDenyConfig   `yaml:"on_deny,omitempty"`
	
	// Metadata fields for documentation and organization (not used in authorization)
	Category    string          `yaml:"category,omitempty"`
	Tags        []string        `yaml:"tags,omitempty"`
}

// InterceptConfig specifies which function to intercept
type InterceptConfig struct {
	Package  string `yaml:"package"`
	Function string `yaml:"function"`
	Pattern  string `yaml:"pattern,omitempty"`
}

// QuotaConfig defines usage quota limits
type QuotaConfig struct {
	Limit     int64  `yaml:"limit"`
	Period    string `yaml:"period"` // daily, hourly, monthly
	ResetTime string `yaml:"reset_time,omitempty"`
}

// ConditionConfig defines conditional feature checking
type ConditionConfig struct {
	Type  string `yaml:"type"`  // runtime, static
	Check string `yaml:"check"` // condition expression
}

// OnDenyConfig specifies behavior when feature is denied
type OnDenyConfig struct {
	Action  string `yaml:"action"`  // fallback, error, warn, filter
	Message string `yaml:"message,omitempty"`
	Code    string `yaml:"error_code,omitempty"`
}

// Validate performs validation on the manifest
func (m *Manifest) Validate() error {
	// Validate SDK config
	if err := m.SDK.Validate(); err != nil {
		return err
	}

	// Validate features
	featureIDs := make(map[string]bool)
	for i, feature := range m.Features {
		if err := feature.Validate(); err != nil {
			return &ValidationError{
				Field:   "features[" + string(rune(i)) + "]",
				Message: err.Error(),
			}
		}

		// Check for duplicate feature IDs
		if featureIDs[feature.ID] {
			return &ValidationError{
				Field:   "features[" + string(rune(i)) + "].id",
				Message: "duplicate feature ID: " + feature.ID,
			}
		}
		featureIDs[feature.ID] = true
	}

	return nil
}

// Validate validates SDK configuration
func (c *SDKConfig) Validate() error {
	if c.LCCURL == "" {
		return &ValidationError{Field: "sdk.lcc_url", Message: "required"}
	}
	if c.ProductID == "" {
		return &ValidationError{Field: "sdk.product_id", Message: "required"}
	}
	if c.ProductVersion == "" {
		return &ValidationError{Field: "sdk.product_version", Message: "required"}
	}

	// Set defaults
	if c.CheckInterval == 0 {
		c.CheckInterval = 30 * time.Second
	}
	if c.CacheTTL == 0 {
		c.CacheTTL = 10 * time.Second
	}
	if c.Timeout == 0 {
		c.Timeout = 5 * time.Second
	}
	if c.MaxRetries == 0 {
		c.MaxRetries = 3
	}

	// Validate product limits if present
	if c.Limits != nil {
		if err := c.Limits.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// Validate validates feature configuration
func (f *FeatureConfig) Validate() error {
	if f.ID == "" {
		return &ValidationError{Field: "id", Message: "required"}
	}
	if f.Name == "" {
		return &ValidationError{Field: "name", Message: "required"}
	}

	// Validate intercept config
	if err := f.Intercept.Validate(); err != nil {
		return &ValidationError{Field: "intercept", Message: err.Error()}
	}

	// Validate fallback if present
	if f.Fallback != nil {
		if err := f.Fallback.Validate(); err != nil {
			return &ValidationError{Field: "fallback", Message: err.Error()}
		}
	}

	// Validate quota if present
	if f.Quota != nil {
		if err := f.Quota.Validate(); err != nil {
			return &ValidationError{Field: "quota", Message: err.Error()}
		}
	}

	// Validate on_deny if present
	if f.OnDeny != nil {
		if err := f.OnDeny.Validate(); err != nil {
			return &ValidationError{Field: "on_deny", Message: err.Error()}
		}
	}

	return nil
}

// Validate validates intercept configuration
func (i *InterceptConfig) Validate() error {
	if i.Package == "" {
		return &ValidationError{Field: "package", Message: "required"}
	}
	if i.Function == "" && i.Pattern == "" {
		return &ValidationError{
			Field:   "function",
			Message: "either function or pattern is required",
		}
	}
	return nil
}

// Validate validates quota configuration
func (q *QuotaConfig) Validate() error {
	if q.Limit <= 0 {
		return &ValidationError{Field: "limit", Message: "must be positive"}
	}

	validPeriods := map[string]bool{
		"daily":   true,
		"hourly":  true,
		"monthly": true,
		"minute":  true,
	}

	if !validPeriods[q.Period] {
		return &ValidationError{
			Field:   "period",
			Message: "must be one of: daily, hourly, monthly, minute",
		}
	}

	return nil
}

// Validate validates on_deny configuration
func (o *OnDenyConfig) Validate() error {
	validActions := map[string]bool{
		"fallback": true,
		"error":    true,
		"warn":     true,
		"filter":   true,
	}

	if !validActions[o.Action] {
		return &ValidationError{
			Field:   "action",
			Message: "must be one of: fallback, error, warn, filter",
		}
	}

	return nil
}

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return "validation error in field '" + e.Field + "': " + e.Message
}

// GetDefaults returns a manifest with default values
func GetDefaults() *Manifest {
	return &Manifest{
		SDK: SDKConfig{
			LCCURL:         "http://localhost:7086",
			CheckInterval:  30 * time.Second,
			CacheTTL:       10 * time.Second,
			FailOpen:       false,
			Timeout:        5 * time.Second,
			MaxRetries:     3,
		},
		Features: []FeatureConfig{},
	}
}

// ProductLimits defines product-level limits for zero-intrusion API.
// These limits apply to the entire product, not individual features.
// This enables the zero-intrusion API where featureID is not required.
type ProductLimits struct {
	// Quota configuration for product-level consumption tracking
	Quota *ProductQuotaConfig `yaml:"quota,omitempty"`

	// MaxTPS defines maximum transactions per second limit
	MaxTPS float64 `yaml:"max_tps,omitempty"`

	// MaxCapacity defines maximum capacity limit (e.g., max users, max connections)
	MaxCapacity int `yaml:"max_capacity,omitempty"`

	// MaxConcurrency defines maximum concurrent operations limit
	MaxConcurrency int `yaml:"max_concurrency,omitempty"`

	// Helper function references (for code generator)
	// These specify which helper functions to call for dynamic behavior

	// Consumer is the name of the QuotaConsumer helper function (optional)
	// Example: "calculateBatchSize"
	Consumer string `yaml:"consumer,omitempty"`

	// TPSProvider is the name of the TPSProvider helper function (optional)
	// Example: "getCurrentTPS"
	TPSProvider string `yaml:"tps_provider,omitempty"`

	// CapacityCounter is the name of the CapacityCounter helper function (required for capacity limits)
	// Example: "countActiveUsers"
	CapacityCounter string `yaml:"capacity_counter,omitempty"`
}

// ProductQuotaConfig defines quota configuration for product-level limits
type ProductQuotaConfig struct {
	// Max is the maximum number of quota units allowed
	Max int `yaml:"max"`

	// Window is the time window for quota calculation
	// Examples: "1h", "24h", "30d"
	Window string `yaml:"window"`
}

// Validate validates product limits configuration
func (p *ProductLimits) Validate() error {
	if p == nil {
		return nil // Product limits are optional
	}

	// Validate quota if present
	if p.Quota != nil {
		if p.Quota.Max <= 0 {
			return &ValidationError{
				Field:   "limits.quota.max",
				Message: "must be positive",
			}
		}
		if p.Quota.Window == "" {
			return &ValidationError{
				Field:   "limits.quota.window",
				Message: "required",
			}
		}
	}

	// Validate numeric limits are non-negative
	if p.MaxTPS < 0 {
		return &ValidationError{
			Field:   "limits.max_tps",
			Message: "must be non-negative",
		}
	}
	if p.MaxCapacity < 0 {
		return &ValidationError{
			Field:   "limits.max_capacity",
			Message: "must be non-negative",
		}
	}
	if p.MaxConcurrency < 0 {
		return &ValidationError{
			Field:   "limits.max_concurrency",
			Message: "must be non-negative",
		}
	}

	// Warn if capacity limit is defined but no counter helper is specified
	if p.MaxCapacity > 0 && p.CapacityCounter == "" {
		// Note: This is a warning, not an error, because the helper can be
		// registered programmatically via RegisterHelpers()
	}

	return nil
}
