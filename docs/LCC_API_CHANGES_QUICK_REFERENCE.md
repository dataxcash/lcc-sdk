# LCC Server API Changes - Quick Reference

**For**: LCC Server Developers  
**SDK Version**: 2.0.0  
**Date**: 2025-01-22

---

## TL;DR

SDK now uses `__product__` as a special feature ID for **product-level limit checks**. LCC Server must:

1. Recognize `__product__` as special feature ID
2. Return product-level limits (not feature-specific)
3. Track quota at product level
4. Maintain backward compatibility with feature-level checks

---

## API Changes Summary

### 1. Feature Check Endpoint

**Endpoint**: `GET /api/v1/sdk/features/{featureID}/check`

#### New Request Pattern

```http
GET /api/v1/sdk/features/__product__/check
```

**When `featureID = "__product__"`**:
- Return product-level limits (not feature-specific)
- Include quota, TPS, capacity, concurrency from product-level config
- Don't look for a feature named "__product__"

#### Response Format (NEW)

```json
{
  "feature_id": "__product__",
  "enabled": true,
  "reason": "ok",
  "quota_info": {
    "limit": 1000,
    "used": 150,
    "remaining": 850,
    "reset_at": 1706054400
  },
  "max_tps": 100.0,
  "max_capacity": 500,
  "max_concurrency": 10,
  "cache_ttl": 30
}
```

### 2. Usage Report Endpoint

**Endpoint**: `POST /api/v1/sdk/usage`

#### New Request Body Pattern

```json
{
  "instance_id": "fingerprint-abc123",
  "feature_id": "__product__",
  "count": 10,
  "timestamp": 1706022000
}
```

**When `feature_id = "__product__"`**:
- Increment product-level quota (not feature-specific)
- Update product-level usage statistics
- Don't attribute to specific feature

---

## License Format Changes

### Add Product-Level Limits Section

```json
{
  "licenseId": "lic-123",
  "productId": "my-app",
  "planInfo": {
    "productLimits": {
      "quota": {
        "max": 1000,
        "window": "24h"
      },
      "maxTPS": 100.0,
      "maxCapacity": 500,
      "maxConcurrency": 10
    },
    "features": {
      "feature-export": {"enabled": true},
      "feature-analytics": {"enabled": true}
    }
  }
}
```

---

## Server Implementation Checklist

### Critical (Required)

- [ ] Handle `feature_id = "__product__"` in feature check endpoint
- [ ] Return product-level limits for `__product__` requests
- [ ] Track product-level quota usage
- [ ] Update usage report handler for product-level
- [ ] Add product-level limits to license structure

### Important (Recommended)

- [ ] Add database schema for product-level limits
- [ ] Create migration script for existing licenses
- [ ] Add validation for product-level requests
- [ ] Implement caching for product limits

### Optional (Nice to Have)

- [ ] Add analytics dashboard for product-level usage
- [ ] Implement quota forecasting
- [ ] Add quota alerts

---

## Code Examples

### Python - Feature Check Handler

```python
def check_feature(feature_id: str, instance_id: str) -> dict:
    license = get_license_for_instance(instance_id)
    
    if feature_id == "__product__":
        # Product-level check
        product_limits = license.get("productLimits", {})
        return {
            "feature_id": "__product__",
            "enabled": license.get("valid", False),
            "reason": "ok" if license.get("valid") else "invalid_license",
            "quota_info": product_limits.get("quota"),
            "max_tps": product_limits.get("maxTPS", 0),
            "max_capacity": product_limits.get("maxCapacity", 0),
            "max_concurrency": product_limits.get("maxConcurrency", 0),
            "cache_ttl": 30
        }
    else:
        # Feature-level check (existing logic)
        feature = license.get("features", {}).get(feature_id)
        return {
            "feature_id": feature_id,
            "enabled": feature.get("enabled", False) if feature else False,
            "reason": "ok" if feature else "feature_not_found",
            # ... other feature-specific fields
        }
```

### Python - Usage Report Handler

```python
def handle_usage_report(instance_id: str, feature_id: str, count: int, timestamp: int):
    if feature_id == "__product__":
        # Product-level usage
        product_id = get_product_id_from_instance(instance_id)
        increment_product_quota(product_id, count)
        log_product_usage(product_id, instance_id, count, timestamp)
    else:
        # Feature-level usage (existing logic)
        increment_feature_quota(feature_id, count)
        log_feature_usage(feature_id, instance_id, count, timestamp)
```

### Go - Feature Check Handler

```go
func checkFeature(featureID, instanceID string) (*FeatureStatus, error) {
    license, err := getLicenseForInstance(instanceID)
    if err != nil {
        return nil, err
    }
    
    if featureID == "__product__" {
        // Product-level check
        productLimits := license.ProductLimits
        return &FeatureStatus{
            FeatureID:      "__product__",
            Enabled:        license.Valid,
            Reason:         "ok",
            QuotaInfo:      productLimits.Quota,
            MaxTPS:         productLimits.MaxTPS,
            MaxCapacity:    productLimits.MaxCapacity,
            MaxConcurrency: productLimits.MaxConcurrency,
            CacheTTL:       30,
        }, nil
    }
    
    // Feature-level check (existing logic)
    feature := license.Features[featureID]
    return &FeatureStatus{
        FeatureID: featureID,
        Enabled:   feature.Enabled,
        // ... other fields
    }, nil
}
```

---

## Database Schema Changes

### Option 1: Extend License Table

```sql
ALTER TABLE licenses ADD COLUMN product_quota_max INT;
ALTER TABLE licenses ADD COLUMN product_quota_used INT;
ALTER TABLE licenses ADD COLUMN product_quota_window VARCHAR(50);
ALTER TABLE licenses ADD COLUMN product_quota_reset_at BIGINT;
ALTER TABLE licenses ADD COLUMN product_max_tps DECIMAL(10,2);
ALTER TABLE licenses ADD COLUMN product_max_capacity INT;
ALTER TABLE licenses ADD COLUMN product_max_concurrency INT;
```

### Option 2: New Product Limits Table

```sql
CREATE TABLE product_limits (
    product_id VARCHAR(255) PRIMARY KEY,
    license_id VARCHAR(255) REFERENCES licenses(id),
    quota_max INT,
    quota_used INT,
    quota_window VARCHAR(50),
    quota_reset_at BIGINT,
    max_tps DECIMAL(10,2),
    max_capacity INT,
    max_concurrency INT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE INDEX idx_product_limits_license ON product_limits(license_id);
CREATE INDEX idx_product_limits_product ON product_limits(product_id);
```

---

## Testing Commands

### Test Product-Level Check

```bash
curl -X GET http://localhost:7086/api/v1/sdk/features/__product__/check \
  -H "X-LCC-Signature: <signature>" \
  -H "X-LCC-Public-Key: <public_key>" \
  -H "X-LCC-Timestamp: <timestamp>"
```

Expected response includes product-level limits.

### Test Product-Level Usage Report

```bash
curl -X POST http://localhost:7086/api/v1/sdk/usage \
  -H "Content-Type: application/json" \
  -H "X-LCC-Signature: <signature>" \
  -d '{
    "instance_id": "fingerprint-abc123",
    "feature_id": "__product__",
    "count": 10,
    "timestamp": 1706022000
  }'
```

Expected: Product quota increases by 10.

---

## Backward Compatibility

**IMPORTANT**: Feature-level checks must continue to work!

```http
GET /api/v1/sdk/features/feature-export/check  ← Still works
GET /api/v1/sdk/features/__product__/check      ← New behavior
```

Both endpoints must coexist during migration period.

---

## Migration Strategy

### Phase 1: Support Both (Recommended)

```
┌─────────────────────────────────┐
│  License with Both Levels       │
├─────────────────────────────────┤
│  productLimits:                 │
│    quota: 1000 (shared)         │
│  features:                      │
│    feature-export: enabled      │
│    feature-analytics: enabled   │
└─────────────────────────────────┘
         ↓              ↓
    Feature-level   Product-level
    Check (old)     Check (new)
```

### Phase 2: Product-Level Only (Future)

Eventually, all licenses can use only product-level limits.

---

## Common Pitfalls

❌ **Don't**: Look for a feature named "__product__"  
✅ **Do**: Treat "__product__" as a special flag for product-level check

❌ **Don't**: Return 404 for "__product__"  
✅ **Do**: Return product-level limits from license

❌ **Don't**: Break feature-level checks  
✅ **Do**: Maintain backward compatibility

---

## Questions?

See detailed documentation:
- [LCC Server Modifications Required](./LCC_SERVER_MODIFICATIONS_REQUIRED.md)
- [SDK Refactoring Completion Report](./REFACTORING_COMPLETION_ZERO_INTRUSION.md)

---

**Last Updated**: 2025-01-22
