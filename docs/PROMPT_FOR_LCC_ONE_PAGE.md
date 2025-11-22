# LCC Server: Zero-Intrusion API Support (One-Page Brief)

**Date**: 2025-01-22 | **Priority**: High | **Effort**: 2-3 weeks

---

## 🎯 Mission

Enable **product-level limit checks** in LCC Server to support lcc-sdk v2.0's zero-intrusion API.

---

## 🔑 Key Concept

SDK now uses `__product__` as a special feature ID to request **product-level limits** instead of feature-specific limits.

---

## 📋 What You Need to Do

### 1. Recognize `__product__` as Special Feature ID

```python
def check_feature(feature_id, instance_id):
    license = get_license_for_instance(instance_id)
    
    if feature_id == "__product__":
        # Return product-level limits
        return {
            "feature_id": "__product__",
            "enabled": license.valid,
            "quota_info": license.productLimits.quota,
            "max_tps": license.productLimits.maxTPS,
            "max_capacity": license.productLimits.maxCapacity,
            "max_concurrency": license.productLimits.maxConcurrency
        }
    else:
        # Feature-level check (keep existing logic)
        return check_feature_level(feature_id, license)
```

### 2. Extend License Format

```json
{
  "licenseId": "lic-123",
  "planInfo": {
    "productLimits": {
      "quota": {"max": 1000, "window": "24h"},
      "maxTPS": 100.0,
      "maxCapacity": 500,
      "maxConcurrency": 10
    },
    "features": {
      "feature-export": {"enabled": true}
    }
  }
}
```

### 3. Track Product-Level Usage

```python
def handle_usage_report(instance_id, feature_id, count):
    if feature_id == "__product__":
        increment_product_quota(get_product_id(instance_id), count)
    else:
        increment_feature_quota(feature_id, count)
```

### 4. Database Schema (Choose One)

**Option A**: Extend licenses table
```sql
ALTER TABLE licenses ADD COLUMN product_quota_max INT;
ALTER TABLE licenses ADD COLUMN product_quota_used INT;
-- ... other product limit columns
```

**Option B**: New table
```sql
CREATE TABLE product_limits (
    product_id VARCHAR(255) PRIMARY KEY,
    license_id VARCHAR(255),
    quota_max INT,
    quota_used INT,
    -- ... other limit columns
);
```

---

## ✅ Success Criteria

- [ ] `GET /api/v1/sdk/features/__product__/check` returns product limits
- [ ] `POST /api/v1/sdk/usage` with `feature_id="__product__"` updates product quota
- [ ] Feature-level checks still work (backward compatibility)
- [ ] All tests pass
- [ ] Deployed to production

---

## ⏱️ Timeline

| Week | Tasks |
|------|-------|
| 1 | Core logic + DB schema |
| 2 | Usage tracking + License migration |
| 3 | Testing + Deployment |

---

## 🚨 Critical: Backward Compatibility

**Both APIs must work!**

```
/features/feature-export/check  ← Old (keep working)
/features/__product__/check     ← New (add support)
```

---

## 📚 Full Documentation

Location: `/home/fila/jqdDev_2025/lcc-sdk/docs/`

**Start here**:
1. `LCC_API_CHANGES_QUICK_REFERENCE.md` - Quick guide with code examples
2. `LCC_SERVER_MODIFICATIONS_REQUIRED.md` - Complete specification
3. `LCC_SDK_INTERACTION_DIAGRAM.md` - Visual flow diagrams

---

## 🧪 Test Commands

```bash
# Test product-level check
curl http://localhost:7086/api/v1/sdk/features/__product__/check

# Test product-level usage
curl -X POST http://localhost:7086/api/v1/sdk/usage \
  -d '{"feature_id": "__product__", "count": 10}'

# Test backward compatibility
curl http://localhost:7086/api/v1/sdk/features/feature-export/check
```

---

## 💡 Key Insights

**What changes**: Recognize `__product__` → Return product limits  
**What stays**: Feature-level API continues working  
**Why**: Enable zero-intrusion license enforcement  

---

## 🔗 Quick Links

- Full prompt: `PROMPT_FOR_LCC_SERVER_TEAM.md`
- API changes: `LCC_API_CHANGES_QUICK_REFERENCE.md`
- Detailed spec: `LCC_SERVER_MODIFICATIONS_REQUIRED.md`
- Diagrams: `LCC_SDK_INTERACTION_DIAGRAM.md`

---

**Questions?** Check the detailed documentation or contact SDK team.

**Ready to start?** Read `LCC_API_CHANGES_QUICK_REFERENCE.md` first! 🚀
