# LCC Server Modifications Required for Zero-Intrusion SDK

**Date**: 2025-01-22  
**SDK Version**: 2.0.0  
**Priority**: High  

---

## Overview

This document describes the modifications required on the **LCC Server** side to support the new **zero-intrusion API** in lcc-sdk v2.0. The SDK has been refactored to use product-level limits instead of feature-level limits, which requires corresponding changes in the LCC server.

---

## Key Changes in SDK

### What Changed

| Aspect | Old Behavior | New Behavior |
|--------|-------------|--------------|
| **Scope** | Feature-level checks | Product-level checks |
| **Feature ID** | Required in every API call | Optional (uses `__product__` for product-level) |
| **Limits** | Per-feature limits | Product-level limits |
| **API Endpoint** | `/api/v1/sdk/features/{featureID}/check` | Same endpoint, but with special `__product__` ID |

### SDK API Changes

**Old SDK API** (still supported):
```go
client.CheckFeature("feature-export")          // Feature-specific
client.ConsumeDeprecated("feature-export", 10, nil)
client.CheckTPSDeprecated("feature-export", currentTPS)
```

**New SDK API** (zero-intrusion):
```go
client.checkProductLimits()                    // Uses "__product__" as featureID
client.Consume(10)                             // Product-level
client.CheckTPS()                              // Product-level
client.CheckCapacity(currentUsed)              // Product-level
```

---

## Required Server-Side Modifications

### 1. Recognize Special Product-Level Feature ID

#### Requirement

LCC server must recognize `__product__` as a special feature ID that indicates a **product-level limit check** rather than a feature-specific check.

#### Implementation

**Endpoint**: `GET /api/v1/sdk/features/__product__/check`

When the server receives a check request with `featureID = "__product__"`:

1. **Don't look for a feature named "__product__"**
2. **Return product-level limits** from the license
3. **Aggregate all product-level quotas** (not feature-specific)

#### Example Request

```http
GET /api/v1/sdk/features/__product__/check HTTP/1.1
Host: localhost:7086
X-LCC-Signature: <signature>
X-LCC-Public-Key: <public_key>
X-LCC-Timestamp: <timestamp>
```

#### Example Response

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
  "max_capacity": 500,
  "max_tps": 100.0,
  "max_concurrency": 10,
  "cache_ttl": 30
}
```

---

### 2. Product-Level Quota Management

#### Requirement

LCC server must track and enforce quota at the **product level**, not per-feature.

#### Current Behavior (Feature-Level)

```json
{
  "features": {
    "feature-export": {
      "quota": {"daily": 500, "used": 100}
    },
    "feature-analytics": {
      "quota": {"daily": 300, "used": 50}
    }
  }
}
```

Total quota is **split across features** (500 + 300 = 800 total).

#### Required Behavior (Product-Level)

```json
{
  "product_limits": {
    "quota": {
      "daily": 1000,
      "used": 150
    },
    "max_tps": 100.0,
    "max_capacity": 500,
    "max_concurrency": 10
  }
}
```

Total quota is **shared across the entire product** (1000 total, used by any feature).

#### Database Schema Changes

**Option A: Add Product-Level Limits Table**

```sql
CREATE TABLE product_limits (
    product_id VARCHAR(255) PRIMARY KEY,
    quota_limit INT,
    quota_used INT,
    quota_window VARCHAR(50),
    quota_reset_at BIGINT,
    max_tps DECIMAL(10,2),
    max_capacity INT,
    max_concurrency INT,
    created_at TIMESTAMP,
    updated_at TIMESTAMP
);
```

**Option B: Extend License Structure**

Add product-level limits to existing license structure:

```json
{
  "licenseId": "lic-123",
  "productId": "my-app",
  "planInfo": {
    "productLimits": {
      "quota": {
        "max": 1000,
        "window": "24h",
        "used": 150,
        "resetAt": 1706054400
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

### 3. Product-Level Usage Reporting

#### Requirement

LCC server must accept and track usage reports at the product level.

#### Current Endpoint

```
POST /api/v1/sdk/usage
```

#### Request Body (Current)

```json
{
  "instance_id": "fingerprint-abc123",
  "feature_id": "feature-export",
  "count": 10,
  "timestamp": 1706022000
}
```

#### Request Body (New - Product Level)

```json
{
  "instance_id": "fingerprint-abc123",
  "feature_id": "__product__",
  "count": 10,
  "timestamp": 1706022000
}
```

#### Required Behavior

When `feature_id = "__product__"`:

1. Increment **product-level quota usage** (not feature-specific)
2. Update `product_limits.quota_used` 
3. Don't attribute usage to any specific feature
4. Update usage statistics for product as a whole

#### Implementation Example

```python
def handle_usage_report(instance_id, feature_id, count, timestamp):
    if feature_id == "__product__":
        # Product-level usage
        product_id = get_product_id_from_instance(instance_id)
        increment_product_quota_usage(product_id, count)
        log_product_usage(product_id, instance_id, count, timestamp)
    else:
        # Feature-level usage (backward compatibility)
        increment_feature_quota_usage(feature_id, count)
        log_feature_usage(feature_id, instance_id, count, timestamp)
```

---

### 4. License File Format Extension

#### Requirement

License files must support product-level limits in addition to feature-level configuration.

#### Current License Format

```json
{
  "licenseId": "lic-123",
  "productId": "my-app",
  "version": "1.0",
  "planInfo": {
    "planName": "Professional",
    "features": {
      "feature-export": {
        "enabled": true,
        "quota": {"daily": 500}
      }
    }
  }
}
```

#### Extended License Format (Required)

```json
{
  "licenseId": "lic-123",
  "productId": "my-app",
  "version": "2.0",
  "planInfo": {
    "planName": "Professional",
    
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
      "feature-export": {
        "enabled": true
      },
      "feature-analytics": {
        "enabled": true
      }
    }
  }
}
```

**Key Changes**:

1. Add `productLimits` section at plan level
2. Remove quota from individual features (optional, for backward compatibility)
3. Features now mainly control enabled/disabled state
4. Quota is shared across all features at product level

---

### 5. API Response Structure

#### Feature Check Response (`__product__`)

When checking `__product__`, the response must include product-level limits:

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
  
  "max_capacity": 500,
  "max_tps": 100.0,
  "max_concurrency": 10,
  "cache_ttl": 30
}
```

**Fields Explanation**:

- `feature_id`: `"__product__"` indicates product-level response
- `enabled`: Whether the license is valid and active
- `quota_info`: Product-level quota information
  - `limit`: Total quota for the product
  - `used`: Quota consumed so far
  - `remaining`: Quota remaining
  - `reset_at`: Unix timestamp when quota resets
- `max_capacity`: Product-level capacity limit (e.g., max users)
- `max_tps`: Product-level TPS limit
- `max_concurrency`: Product-level concurrency limit

---

### 6. Backward Compatibility

#### Requirement

LCC server must support **both** old feature-level and new product-level APIs simultaneously.

#### Strategy

**For Feature-Level Checks** (existing behavior):
```
GET /api/v1/sdk/features/feature-export/check
```
Response includes feature-specific limits (if defined in license).

**For Product-Level Checks** (new behavior):
```
GET /api/v1/sdk/features/__product__/check
```
Response includes product-level limits (if defined in license).

#### Decision Logic

```python
def check_feature(feature_id, instance_id):
    license = get_license_for_instance(instance_id)
    
    if feature_id == "__product__":
        # Product-level check
        return {
            "feature_id": "__product__",
            "enabled": license.is_valid(),
            "quota_info": license.product_limits.quota,
            "max_tps": license.product_limits.max_tps,
            "max_capacity": license.product_limits.max_capacity,
            "max_concurrency": license.product_limits.max_concurrency,
        }
    else:
        # Feature-level check (backward compatibility)
        feature = license.features.get(feature_id)
        return {
            "feature_id": feature_id,
            "enabled": feature.enabled if feature else False,
            "quota_info": feature.quota if feature else None,
            # ... other feature-specific limits
        }
```

---

### 7. Configuration Migration

#### For Existing Licenses

When migrating existing licenses to support product-level limits:

**Option A: Aggregate Feature Quotas**
```python
def migrate_license_to_product_level(license):
    total_quota = sum(feature.quota for feature in license.features.values())
    license.product_limits = {
        "quota": {"max": total_quota, "window": "24h"}
    }
```

**Option B: Set New Product Quota**
```python
def migrate_license_to_product_level(license):
    # Define product-level quota based on plan
    plan_quotas = {
        "basic": 500,
        "professional": 1000,
        "enterprise": 5000
    }
    license.product_limits = {
        "quota": {"max": plan_quotas[license.plan], "window": "24h"}
    }
```

#### For New Licenses

New licenses should be created with product-level limits from the start:

```json
{
  "productLimits": {
    "quota": {"max": 1000, "window": "24h"},
    "maxTPS": 100.0,
    "maxCapacity": 500,
    "maxConcurrency": 10
  }
}
```

---

## Implementation Checklist

### Phase 1: Core Functionality

- [ ] Recognize `__product__` as special feature ID
- [ ] Add product-level limits to license structure
- [ ] Implement product-level quota tracking
- [ ] Return product limits in API response
- [ ] Handle product-level usage reports

### Phase 2: Database Changes

- [ ] Add product-level limits table/fields
- [ ] Create migration script for existing licenses
- [ ] Add indexes for performance
- [ ] Update quota tracking queries

### Phase 3: API Updates

- [ ] Update `/api/v1/sdk/features/__product__/check` endpoint
- [ ] Update `/api/v1/sdk/usage` endpoint for product-level
- [ ] Add validation for product-level requests
- [ ] Update error messages

### Phase 4: Testing

- [ ] Test product-level feature checks
- [ ] Test quota consumption and tracking
- [ ] Test backward compatibility with feature-level
- [ ] Test quota reset logic
- [ ] Load testing for performance

### Phase 5: Documentation

- [ ] Update API documentation
- [ ] Create migration guide for license administrators
- [ ] Update license file format documentation
- [ ] Add examples and tutorials

---

## Testing Scenarios

### Scenario 1: Product-Level Quota Check

**Request**:
```http
GET /api/v1/sdk/features/__product__/check
```

**Expected Response**:
```json
{
  "feature_id": "__product__",
  "enabled": true,
  "quota_info": {
    "limit": 1000,
    "used": 0,
    "remaining": 1000,
    "reset_at": 1706054400
  }
}
```

### Scenario 2: Product-Level Usage Report

**Request**:
```http
POST /api/v1/sdk/usage
Content-Type: application/json

{
  "instance_id": "fingerprint-abc123",
  "feature_id": "__product__",
  "count": 10,
  "timestamp": 1706022000
}
```

**Expected**: Quota used increases to 10

### Scenario 3: Quota Exceeded

**When**: Product quota = 1000, used = 995

**Request**: Report usage of 10 units

**Expected Response on next check**:
```json
{
  "feature_id": "__product__",
  "enabled": false,
  "reason": "quota_exceeded",
  "quota_info": {
    "limit": 1000,
    "used": 1005,
    "remaining": 0,
    "reset_at": 1706054400
  }
}
```

### Scenario 4: Backward Compatibility

**Old SDK Request**:
```http
GET /api/v1/sdk/features/feature-export/check
```

**Expected**: Feature-level check still works as before

---

## Performance Considerations

### Database Queries

**Optimize product-level quota lookups**:
```sql
-- Add index for fast product-level queries
CREATE INDEX idx_product_limits_product_id ON product_limits(product_id);

-- Efficient quota update
UPDATE product_limits 
SET quota_used = quota_used + 10 
WHERE product_id = 'my-app';
```

### Caching

Product-level limits should be cached aggressively since they change less frequently than feature-level:

```python
PRODUCT_LIMITS_CACHE_TTL = 60  # seconds

def get_product_limits(product_id):
    cache_key = f"product_limits:{product_id}"
    limits = cache.get(cache_key)
    if limits is None:
        limits = db.get_product_limits(product_id)
        cache.set(cache_key, limits, ttl=PRODUCT_LIMITS_CACHE_TTL)
    return limits
```

---

## Security Considerations

1. **Validate Feature ID**:
   - Ensure `__product__` is only used for product-level checks
   - Prevent injection attacks with special characters

2. **Rate Limiting**:
   - Apply rate limits to prevent abuse of product-level endpoints
   - Track usage patterns for anomaly detection

3. **Authorization**:
   - Verify instance signature for all requests
   - Ensure product-level checks are authorized

---

## Migration Plan

### Step 1: Add Product Limits Support (Week 1)

- Add database schema
- Implement product-level endpoints
- Add backward compatibility layer

### Step 2: Test with SDK v2.0 (Week 2)

- Deploy to test environment
- Run integration tests with new SDK
- Verify backward compatibility

### Step 3: Migrate Existing Licenses (Week 3)

- Run migration scripts
- Update licenses with product-level limits
- Validate migration results

### Step 4: Production Rollout (Week 4)

- Deploy to production
- Monitor logs and metrics
- Support customer migration

---

## Summary

### Critical Changes

1. ✅ **Recognize `__product__` feature ID**
2. ✅ **Add product-level limits to license structure**
3. ✅ **Track quota at product level**
4. ✅ **Support product-level usage reports**
5. ✅ **Maintain backward compatibility**

### Optional Enhancements

- [ ] Add analytics for product-level usage
- [ ] Implement quota forecasting
- [ ] Add quota alerts and notifications
- [ ] Create admin UI for product limits

---

## Questions?

For implementation questions:
- Review SDK source: `/home/fila/jqdDev_2025/lcc-sdk/pkg/client/client.go`
- Check SDK documentation: `/home/fila/jqdDev_2025/lcc-sdk/docs/`
- Test with examples: `/home/fila/jqdDev_2025/lcc-sdk/examples/zero-intrusion/`

---

**Last Updated**: 2025-01-22  
**Document Version**: 1.0
