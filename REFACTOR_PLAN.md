# LCC-SDK Refactoring Plan

## Problem Statement

Current architecture mixes **feature definition** (YAML) with **authorization control** (License):
- YAML defines `tier` requirement (e.g., "professional")
- License just validates tier match
- **Result**: License has no real control, YAML dictates authorization

## Goal

Separate concerns:
- **YAML**: Feature registration (technical mapping)
- **License**: Authorization control (business decision)

## Architecture Changes

### Before (Current)

```
YAML (lcc-features.yaml):
  - id: advanced_analytics
    tier: professional        ← Authorization logic in YAML!
    intercept: RunAdvanced

License (license.lic):
  planType: professional      ← Just validates tier match
  features: [...]             ← Not used effectively

SDK Check:
  1. Read YAML: needs "professional"
  2. Check License: is tier >= "professional"?
  3. Result: YAML controls, License validates
```

### After (Target)

```
YAML (lcc-features.yaml):
  - id: advanced_analytics
    intercept: RunAdvanced    ← Only function mapping
    fallback: RunBasic
    # NO tier field!

License (license.lic):
  features: {
    "advanced_analytics": {
      "enabled": true,        ← License controls!
      "quota": {...}
    }
  }

SDK Check:
  1. Read YAML: map feature ID to function
  2. Check License: is "advanced_analytics" enabled?
  3. Result: License controls, YAML just maps
```

---

## Detailed Modifications

### 1. YAML Schema Changes

**File**: `lcc-sdk/pkg/config/types.go`

#### Remove `tier` field from FeatureConfig

```go
// BEFORE
type FeatureConfig struct {
    ID          string          `yaml:"id"`
    Name        string          `yaml:"name"`
    Tier        string          `yaml:"tier"`  // ← REMOVE THIS
    Intercept   InterceptConfig `yaml:"intercept"`
    Fallback    *InterceptConfig `yaml:"fallback,omitempty"`
    Quota       *QuotaConfig    `yaml:"quota,omitempty"`
    OnDeny      *OnDenyConfig   `yaml:"on_deny,omitempty"`
}

// AFTER
type FeatureConfig struct {
    ID          string          `yaml:"id"`
    Name        string          `yaml:"name"`
    Description string          `yaml:"description,omitempty"`
    Intercept   InterceptConfig `yaml:"intercept"`
    Fallback    *InterceptConfig `yaml:"fallback,omitempty"`
    OnDeny      *OnDenyConfig   `yaml:"on_deny,omitempty"`
    
    // Metadata for documentation only (not used in authorization)
    Category    string          `yaml:"category,omitempty"`
    Tags        []string        `yaml:"tags,omitempty"`
}
```

#### Remove quota from YAML (move to License)

```go
// Remove QuotaConfig from FeatureConfig
// Quota limits should be defined in License, not YAML
```

**Rationale**:
- YAML is for **technical mapping** (feature ID → function)
- License is for **business rules** (enabled/disabled, quotas, limits)

---

### 2. License Schema Changes

**File**: `lcc/models/sdk.go`

#### Enhance SDKLicenseInfo structure

```go
// BEFORE
type SDKLicenseInfo struct {
    ProductID string
    Version   string
    Tier      string        // Generic tier (deprecated)
    Features  []string      // Just a list (not detailed)
    ExpiresAt time.Time
    
    QuotaLimits map[string]int
    MaxCapacity map[string]int
    MaxTPS map[string]float64
    MaxConcurrency map[string]int
}

// AFTER
type SDKLicenseInfo struct {
    ProductID string
    Version   string
    ExpiresAt time.Time
    
    // Main feature control
    Features  map[string]*FeaturePermission  // ← Change to detailed map
    
    // Deprecated (for backward compatibility)
    Tier      string  `json:"tier,omitempty"`
}

// New structure for per-feature permissions
type FeaturePermission struct {
    Enabled    bool                   `json:"enabled"`
    Quota      *QuotaLimit            `json:"quota,omitempty"`
    Capacity   *CapacityLimit         `json:"capacity,omitempty"`
    RateLimit  *RateLimit             `json:"rate_limit,omitempty"`
    Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

type QuotaLimit struct {
    Daily   int `json:"daily,omitempty"`
    Monthly int `json:"monthly,omitempty"`
    Total   int `json:"total,omitempty"`
}

type CapacityLimit struct {
    MaxCount int `json:"max_count,omitempty"`
}

type RateLimit struct {
    TPS         float64 `json:"tps,omitempty"`
    Concurrency int     `json:"concurrency,omitempty"`
}
```

**Example License JSON**:

```json
{
  "base": {
    "licID": "LIC-12345",
    "expireTime": 1788868632
  },
  "productInfo": {
    "spID": "demo-analytics-pro",
    "productName": "Analytics Pro"
  },
  "planInfo": {
    "features": {
      "advanced_analytics": {
        "enabled": true,
        "quota": {
          "daily": 10000
        },
        "rate_limit": {
          "tps": 100
        }
      },
      "pdf_export": {
        "enabled": true,
        "quota": {
          "daily": 200
        }
      },
      "excel_export": {
        "enabled": false
      }
    }
  }
}
```

---

### 3. SDK Client Changes

**File**: `lcc-sdk/pkg/client/client.go`

#### Update CheckFeature logic

```go
// BEFORE - checks tier from YAML against License tier
func (c *Client) CheckFeature(featureID string) (*FeatureStatus, error) {
    // Query LCC with tier from YAML
    status, err := c.queryFeature(featureID)
    // ...
}

// AFTER - only checks if feature is enabled in License
func (c *Client) CheckFeature(featureID string) (*FeatureStatus, error) {
    // Check cache first
    if status := c.cache.get(featureID); status != nil {
        return status, nil
    }
    
    // Query LCC (no tier parameter needed)
    status, err := c.queryFeature(featureID)
    if err != nil {
        return nil, err
    }
    
    c.cache.set(featureID, status)
    return status, nil
}

// Remove tier parameter from queryFeature
func (c *Client) queryFeature(featureID string) (*FeatureStatus, error) {
    url := fmt.Sprintf("%s/api/v1/sdk/features/%s/check", c.baseURL, featureID)
    // No tier parameter needed
    // ...
}
```

---

### 4. LCC Server Changes

**File**: `lcc/controllers/gin_sdk.go`

#### Update feature check endpoint

```go
// BEFORE - line 164-168
featureTier := c.DefaultQuery("tier", "basic")  // ← REMOVE
enabled, reason := storage.CheckFeatureAuthorized(instance.ProductID, featureID, featureTier)

// AFTER
enabled, reason, featurePermission := storage.CheckFeatureEnabled(instance.ProductID, featureID)

// Return detailed permission info
response := models.SDKFeatureCheckResponse{
    FeatureID:  featureID,
    Enabled:    enabled,
    Reason:     reason,
    
    // From License, not YAML
    Quota:      featurePermission.Quota,
    Capacity:   featurePermission.Capacity,
    RateLimit:  featurePermission.RateLimit,
    
    CacheTTL:   10,
}
```

**File**: `lcc/models/sdk_storage.go`

#### Update CheckFeatureAuthorized to CheckFeatureEnabled

```go
// BEFORE - line 138-174
func (s *SDKStorage) CheckFeatureAuthorized(productID, featureID string, tier string) (bool, string) {
    license, exists := s.licenses[productID]
    if !exists {
        return false, "no_license"
    }
    
    // Check tier-based authorization
    tierLevel := map[string]int{
        "basic": 1,
        "professional": 2,
        "enterprise": 3,
    }
    
    licenseTierLevel := tierLevel[license.Tier]
    featureTierLevel := tierLevel[tier]
    
    if featureTierLevel <= licenseTierLevel {
        return true, "tier_match"
    }
    
    return false, "insufficient_tier"
}

// AFTER
func (s *SDKStorage) CheckFeatureEnabled(productID, featureID string) (bool, string, *models.FeaturePermission) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    
    license, exists := s.licenses[productID]
    if !exists {
        return false, "no_license", nil
    }
    
    if time.Now().After(license.ExpiresAt) {
        return false, "license_expired", nil
    }
    
    // Check if feature is enabled in license
    permission, exists := license.Features[featureID]
    if !exists {
        return false, "feature_not_in_license", nil
    }
    
    if !permission.Enabled {
        return false, "feature_disabled", nil
    }
    
    return true, "ok", permission
}
```

---

### 5. Code Generator Updates

**File**: `lcc-sdk/pkg/codegen/generator.go`

#### Update generated wrapper code

```go
// Generated wrapper should not reference tier
// BEFORE
/*
func RunAdvanced_Wrapped() error {
    // Check if user has professional tier
    status, err := lccClient.CheckFeature("advanced_analytics")
    // ...
}
*/

// AFTER
/*
func RunAdvanced_Wrapped() error {
    // Simply check if feature is enabled
    status, err := lccClient.CheckFeature("advanced_analytics")
    if err != nil {
        return err
    }
    
    if !status.Enabled {
        // Use fallback if configured
        if hasFallback {
            return RunBasic()
        }
        return fmt.Errorf("feature not enabled: %s", status.Reason)
    }
    
    return RunAdvanced_Original()
}
*/
```

---

## Migration Strategy

### Phase 1: Add New Fields (Backward Compatible)

1. Add `Features map[string]*FeaturePermission` to `SDKLicenseInfo`
2. Keep old `Tier` and `Features []string` fields (deprecated)
3. SDK checks new format first, falls back to old format

### Phase 2: Update LMF to Generate New License Format

1. LMF reads product features from manifest
2. LMF generates License with detailed `features` map
3. Old licenses still work via backward compatibility

### Phase 3: Update SDK and Demo Apps

1. Remove `tier` from YAML files
2. Update SDK client to use new API
3. Test with both old and new licenses

### Phase 4: Deprecate Old Format

1. Mark `tier` field as deprecated
2. Update documentation
3. Eventually remove old code path

---

## Testing Plan

### Unit Tests

```go
// lcc/models/sdk_storage_test.go
func TestCheckFeatureEnabled(t *testing.T) {
    storage := GetSDKStorage()
    
    // Setup license with detailed features
    storage.licenses["test-product"] = &SDKLicenseInfo{
        ProductID: "test-product",
        ExpiresAt: time.Now().Add(365 * 24 * time.Hour),
        Features: map[string]*FeaturePermission{
            "advanced_analytics": {
                Enabled: true,
                Quota: &QuotaLimit{Daily: 10000},
            },
            "excel_export": {
                Enabled: false,
            },
        },
    }
    
    // Test enabled feature
    enabled, reason, perm := storage.CheckFeatureEnabled("test-product", "advanced_analytics")
    assert.True(t, enabled)
    assert.Equal(t, "ok", reason)
    assert.NotNil(t, perm)
    assert.Equal(t, 10000, perm.Quota.Daily)
    
    // Test disabled feature
    enabled, reason, _ = storage.CheckFeatureEnabled("test-product", "excel_export")
    assert.False(t, enabled)
    assert.Equal(t, "feature_disabled", reason)
    
    // Test missing feature
    enabled, reason, _ = storage.CheckFeatureEnabled("test-product", "unknown_feature")
    assert.False(t, enabled)
    assert.Equal(t, "feature_not_in_license", reason)
}
```

### Integration Tests

1. Generate test licenses with new format
2. Run demo app with new YAML (no tier field)
3. Verify feature checks work correctly
4. Test quota enforcement
5. Test backward compatibility with old licenses

---

## Documentation Updates

### 1. Update README.md

```markdown
## Feature Definition (YAML)

Define which functions to protect:

```yaml
features:
  - id: advanced_analytics
    name: "Advanced Analytics"
    description: "ML-powered analytics features"
    intercept:
      package: "myapp/analytics"
      function: "RunAdvanced"
    fallback:
      function: "RunBasic"
```

**Note**: Do NOT define `tier` or `quota` in YAML. These are business
decisions controlled by the License file.

## License File

The license file controls which features are enabled:

```json
{
  "planInfo": {
    "features": {
      "advanced_analytics": {
        "enabled": true,
        "quota": {"daily": 10000}
      }
    }
  }
}
```
```

### 2. Update API Documentation

```markdown
## GET /api/v1/sdk/features/:featureId/check

Check if a feature is enabled.

**Request**: No parameters needed (tier removed)

**Response**:
```json
{
  "feature_id": "advanced_analytics",
  "enabled": true,
  "reason": "ok",
  "quota": {
    "daily": 10000,
    "used": 150,
    "remaining": 9850
  }
}
```
```

---

## Benefits of Refactoring

1. **Clear Separation of Concerns**
   - YAML: Technical mapping (feature → function)
   - License: Business rules (enabled/disabled, limits)

2. **Flexible License Control**
   - Can enable/disable individual features
   - Can set different quotas per feature
   - No need to recompile app to change authorization

3. **Better Product Tiers**
   - "Professional" tier = License with specific features enabled
   - Easy to create custom tiers without code changes

4. **Simplified Development**
   - Developers just register features in YAML
   - Sales/License team controls what customers get

5. **Future-Proof**
   - Easy to add new permission types
   - Can support feature flags, A/B testing
   - Supports fine-grained control per customer

---

## Timeline Estimate

- **Phase 1** (Add new fields): 2 days
- **Phase 2** (Update LMF): 3 days
- **Phase 3** (Update SDK/Apps): 3 days
- **Phase 4** (Deprecation): 1 day
- **Testing**: 3 days
- **Documentation**: 2 days

**Total**: ~2 weeks

---

## Summary

**Current Problem**: YAML defines tier requirements, License just validates
**Solution**: Remove tier from YAML, make License the source of truth
**Result**: License has full control, YAML just maps features to functions

This aligns with the design principle:
- **Feature ID** = Stable business interface
- **Function** = Flexible technical implementation
- **License** = Runtime authorization control
