# Prompt for LCC Server Team: Zero-Intrusion API Support

**Date**: 2025-01-22  
**Priority**: High  
**Estimated Effort**: 2-3 weeks  

---

## Task Overview

Implement support for **product-level limit checks** in LCC Server to support the new zero-intrusion API in lcc-sdk v2.0.

---

## Background Context

The lcc-sdk has been refactored to support a **zero-intrusion API** design where:
- Limits are enforced at **product level** (not per-feature)
- Business code remains clean without license checks
- SDK uses `__product__` as a special feature ID for product-level checks

**Your task**: Update LCC Server to recognize and handle product-level limit requests.

---

## Core Requirements

### 1. Recognize Special Feature ID: `__product__`

When the SDK requests feature check with `featureID = "__product__"`, the server must:

- **NOT** look for a feature named "__product__"
- **RETURN** product-level limits from the license
- **AGGREGATE** quota across entire product (not per-feature)

### 2. License Format Extension

Add `productLimits` section to license structure:

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

### 3. API Endpoint Behavior

**Endpoint**: `GET /api/v1/sdk/features/{featureID}/check`

**When `featureID = "__product__"`**:

Return product-level response:
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

### 4. Product-Level Usage Tracking

**Endpoint**: `POST /api/v1/sdk/usage`

**When `feature_id = "__product__"`** in request body:

```json
{
  "instance_id": "fingerprint-abc123",
  "feature_id": "__product__",
  "count": 10,
  "timestamp": 1706022000
}
```

- Increment **product-level quota** (not feature-specific)
- Update `productLimits.quota.used` by `count`
- Track usage for entire product, not individual features

### 5. Backward Compatibility (CRITICAL)

**MUST maintain support for feature-level checks**:

```
GET /api/v1/sdk/features/feature-export/check  ← Old API (keep working)
GET /api/v1/sdk/features/__product__/check      ← New API (add support)
```

Both must coexist!

---

## Implementation Steps

### Phase 1: Core Logic (Week 1)

**Tasks**:
1. Add handler logic for `feature_id == "__product__"`
2. Implement product-level limit retrieval from license
3. Return proper response format
4. Handle product-level usage reports
5. Write unit tests

**Code Example** (Python):

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
        # Feature-level check (existing logic - keep as-is)
        feature = license.get("features", {}).get(feature_id)
        return {
            "feature_id": feature_id,
            "enabled": feature.get("enabled", False) if feature else False,
            "reason": "ok" if feature else "feature_not_found",
            # ... existing fields
        }
```

### Phase 2: Database Schema (Week 1-2)

**Option A: Extend License Table**

```sql
ALTER TABLE licenses ADD COLUMN product_quota_max INT;
ALTER TABLE licenses ADD COLUMN product_quota_used INT;
ALTER TABLE licenses ADD COLUMN product_quota_window VARCHAR(50);
ALTER TABLE licenses ADD COLUMN product_quota_reset_at BIGINT;
ALTER TABLE licenses ADD COLUMN product_max_tps DECIMAL(10,2);
ALTER TABLE licenses ADD COLUMN product_max_capacity INT;
ALTER TABLE licenses ADD COLUMN product_max_concurrency INT;
```

**Option B: New Product Limits Table**

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
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_product_limits_license ON product_limits(license_id);
CREATE INDEX idx_product_limits_product ON product_limits(product_id);
```

**Tasks**:
1. Choose schema approach (Option A or B)
2. Create migration script
3. Add database queries for product limits
4. Add indexes for performance
5. Test migrations

### Phase 3: Usage Tracking (Week 2)

**Update usage report handler**:

```python
def handle_usage_report(instance_id: str, feature_id: str, count: int, timestamp: int):
    if feature_id == "__product__":
        # Product-level usage
        product_id = get_product_id_from_instance(instance_id)
        
        # Increment product quota
        db.execute("""
            UPDATE product_limits 
            SET quota_used = quota_used + %s 
            WHERE product_id = %s
        """, (count, product_id))
        
        # Log usage
        log_product_usage(product_id, instance_id, count, timestamp)
    else:
        # Feature-level usage (existing logic - keep as-is)
        increment_feature_quota(feature_id, count)
        log_feature_usage(feature_id, instance_id, count, timestamp)
```

**Tasks**:
1. Implement product-level quota increment
2. Add product-level usage logging
3. Implement quota reset logic
4. Add validation
5. Write tests

### Phase 4: License Migration (Week 2-3)

**Migrate existing licenses to include product-level limits**:

```python
def migrate_license_to_product_level(license_id: str):
    license = get_license(license_id)
    
    # Strategy: Aggregate feature quotas
    total_quota = sum(
        feature.get("quota", {}).get("daily", 0) 
        for feature in license.get("features", {}).values()
    )
    
    # Create product limits
    product_limits = {
        "quota": {
            "max": total_quota,
            "window": "24h"
        },
        "maxTPS": 0,  # Set based on plan
        "maxCapacity": 0,  # Set based on plan
        "maxConcurrency": 0  # Set based on plan
    }
    
    # Update license
    license["productLimits"] = product_limits
    update_license(license_id, license)
```

**Tasks**:
1. Write migration script
2. Test on sample licenses
3. Run migration on staging
4. Validate results
5. Document rollback procedure

### Phase 5: Testing & Deployment (Week 3)

**Tasks**:
1. Unit tests for all new logic
2. Integration tests with lcc-sdk v2.0
3. Performance testing
4. Test backward compatibility
5. Deploy to staging
6. Deploy to production
7. Monitor metrics

---

## Testing Checklist

### Functional Tests

- [ ] Product-level feature check returns correct format
- [ ] Quota tracking works at product level
- [ ] Usage reports update product quota
- [ ] Quota exceeds correctly
- [ ] Quota resets work
- [ ] Feature-level checks still work (backward compatibility)

### API Tests

```bash
# Test 1: Product-level check
curl -X GET http://localhost:7086/api/v1/sdk/features/__product__/check \
  -H "X-LCC-Signature: <sig>" \
  -H "X-LCC-Public-Key: <key>" \
  -H "X-LCC-Timestamp: <ts>"

# Expected: Returns product-level limits

# Test 2: Product-level usage
curl -X POST http://localhost:7086/api/v1/sdk/usage \
  -H "Content-Type: application/json" \
  -d '{"instance_id": "test-123", "feature_id": "__product__", "count": 10}'

# Expected: Product quota increases by 10

# Test 3: Backward compatibility
curl -X GET http://localhost:7086/api/v1/sdk/features/feature-export/check \
  -H "X-LCC-Signature: <sig>" \
  -H "X-LCC-Public-Key: <key>" \
  -H "X-LCC-Timestamp: <ts>"

# Expected: Feature-level check still works
```

### Performance Tests

- [ ] Product-level queries are fast (<50ms)
- [ ] Quota updates are atomic
- [ ] Caching works correctly
- [ ] Database indexes are used
- [ ] Load test passes (1000 req/s)

---

## Reference Documentation

All detailed documentation is available in the lcc-sdk repository:

### Location: `/home/fila/jqdDev_2025/lcc-sdk/docs/`

**Primary Documents**:

1. **LCC_SERVER_MODIFICATIONS_REQUIRED.md**
   - Complete implementation specification
   - Database schema details
   - Migration strategies
   - Security considerations
   - Performance optimization

2. **LCC_API_CHANGES_QUICK_REFERENCE.md**
   - Quick reference guide
   - Code examples (Python/Go)
   - Common pitfalls
   - Testing commands

3. **LCC_SDK_INTERACTION_DIAGRAM.md**
   - Visual flow diagrams
   - Request/response examples
   - Comparison with old API
   - Migration paths

**Supporting Documents**:

4. **REFACTORING_COMPLETION_ZERO_INTRUSION.md**
   - SDK changes overview
   - Benefits and rationale

5. **MIGRATION_GUIDE_ZERO_INTRUSION.md**
   - How customers will migrate
   - API comparison

---

## Success Criteria

### Must Have (P0)

✅ Recognize `__product__` as special feature ID  
✅ Return product-level limits for `__product__` checks  
✅ Track quota at product level  
✅ Maintain backward compatibility  
✅ All tests pass  

### Should Have (P1)

✅ Database schema implemented  
✅ Migration script for existing licenses  
✅ Performance optimizations (caching, indexes)  
✅ Monitoring and logging  

### Nice to Have (P2)

✅ Admin UI for product limits  
✅ Quota forecasting  
✅ Usage analytics dashboard  

---

## Timeline

```
Week 1:  Core logic + Database schema
Week 2:  Usage tracking + License migration
Week 3:  Testing + Deployment
```

**Milestones**:
- End of Week 1: Core functionality working in dev
- End of Week 2: Migration complete, ready for staging
- End of Week 3: Deployed to production

---

## Common Pitfalls to Avoid

❌ **Don't**: Search for a feature named "__product__" in the features list  
✅ **Do**: Treat "__product__" as a flag to return product-level limits

❌ **Don't**: Return 404 for "__product__"  
✅ **Do**: Return product limits from license.productLimits

❌ **Don't**: Break existing feature-level API  
✅ **Do**: Ensure both APIs work side-by-side

❌ **Don't**: Mix product-level and feature-level quota tracking  
✅ **Do**: Keep them completely separate

---

## Code Review Checklist

Before merging:

- [ ] `__product__` special case is handled
- [ ] Product limits are returned correctly
- [ ] Backward compatibility maintained
- [ ] Database migrations tested
- [ ] Unit tests added
- [ ] Integration tests pass
- [ ] Performance benchmarks meet targets
- [ ] Documentation updated
- [ ] Logging added for debugging
- [ ] Error handling is robust

---

## Support & Questions

**For SDK questions**:
- Repository: `/home/fila/jqdDev_2025/lcc-sdk/`
- Documentation: `/home/fila/jqdDev_2025/lcc-sdk/docs/`
- Examples: `/home/fila/jqdDev_2025/lcc-sdk/examples/zero-intrusion/`

**For implementation questions**:
- Review detailed spec: `LCC_SERVER_MODIFICATIONS_REQUIRED.md`
- Check quick reference: `LCC_API_CHANGES_QUICK_REFERENCE.md`
- Study diagrams: `LCC_SDK_INTERACTION_DIAGRAM.md`

**For testing**:
- Test with lcc-sdk v2.0 examples
- Use provided curl commands
- Follow testing checklist above

---

## Summary

**What you're building**: Support for product-level limit checks in LCC Server

**Why**: Enable zero-intrusion license enforcement in applications

**Key change**: Recognize `__product__` as special feature ID and return product-level limits

**Timeline**: 2-3 weeks

**Priority**: High (required for lcc-sdk v2.0 rollout)

---

## Getting Started

1. **Read documentation** in `/home/fila/jqdDev_2025/lcc-sdk/docs/`:
   - Start with `LCC_API_CHANGES_QUICK_REFERENCE.md`
   - Review `LCC_SERVER_MODIFICATIONS_REQUIRED.md`
   - Study `LCC_SDK_INTERACTION_DIAGRAM.md`

2. **Set up environment**:
   - Clone/update LCC server repository
   - Set up test database
   - Install lcc-sdk v2.0 for testing

3. **Start with Phase 1**:
   - Implement `__product__` recognition
   - Return product-level limits
   - Write basic tests

4. **Follow the phases** outlined above

5. **Test continuously** with lcc-sdk v2.0 examples

---

**Good luck!** 🚀

The SDK team is available for questions and clarifications.

---

**Document Version**: 1.0  
**Last Updated**: 2025-01-22  
**Author**: LCC SDK Team  
**Status**: Ready for Implementation
