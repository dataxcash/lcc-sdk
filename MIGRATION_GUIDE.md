# Migration Guide: Old to New Authorization Model

## Overview

This guide helps you migrate from the old tier-based authorization model to the new license-controlled model.

## What Changed?

### Old Model (Deprecated)

```yaml
# lcc-features.yaml
features:
  - id: advanced_analytics
    tier: professional        # ← YAML controls authorization
    quota:
      limit: 10000
      period: daily
    intercept:
      function: "RunAdvanced"
```

**Problem**: Authorization logic is in YAML, which is a technical configuration file. License just validates tier match.

### New Model (Recommended)

```yaml
# lcc-features.yaml
features:
  - id: advanced_analytics
    # No tier or quota here!
    intercept:
      function: "RunAdvanced"
    fallback:
      function: "RunBasic"
```

```json
// license.lic (decrypted)
{
  "planInfo": {
    "features": {
      "advanced_analytics": {
        "enabled": true,        // ← License controls authorization
        "quota": {"daily": 10000}
      }
    }
  }
}
```

**Benefit**: License has full control. Authorization is a business decision, not a technical one.

---

## Migration Steps

### Step 1: Update YAML Files

**Remove `tier` and `quota` fields from your features:**

```yaml
# BEFORE
features:
  - id: advanced_analytics
    name: "Advanced Analytics"
    tier: professional          # ← Remove
    intercept:
      package: "myapp/analytics"
      function: "AdvancedAnalytics"
    quota:                       # ← Remove
      limit: 10000
      period: daily

# AFTER
features:
  - id: advanced_analytics
    name: "Advanced Analytics"
    description: "ML-powered analytics"  # ← Add description
    category: "Analytics"                 # ← Optional: add metadata
    tags: ["ml", "premium"]               # ← Optional: add tags
    intercept:
      package: "myapp/analytics"
      function: "AdvancedAnalytics"
    fallback:
      package: "myapp/analytics"
      function: "BasicAnalytics"
```

### Step 2: Ensure License Format Compatibility

Your LCC server should return licenses with the new format. The SDK is backward compatible and will work with both formats.

**Old License Format** (still works):
```json
{
  "planInfo": {
    "planType": "professional",
    "features": ["advanced_analytics", "pdf_export"]
  }
}
```

**New License Format** (recommended):
```json
{
  "planInfo": {
    "features": {
      "advanced_analytics": {
        "enabled": true,
        "quota": {"daily": 10000},
        "rate_limit": {"tps": 100}
      },
      "pdf_export": {
        "enabled": true,
        "quota": {"daily": 200}
      },
      "excel_export": {
        "enabled": false
      }
    }
  }
}
```

### Step 3: Regenerate Code

```bash
# Regenerate wrappers with updated YAML
make generate

# Or directly
lcc-codegen --config lcc-features.yaml --output ./
```

### Step 4: Test

```bash
# Run tests to ensure everything works
make test

# Test with old licenses (should still work)
# Test with new licenses (recommended)
```

### Step 5: Update Documentation

Update your internal documentation to reflect:
- Feature IDs are the stable business interface
- Authorization is controlled by License file
- YAML is only for technical mapping

---

## Backward Compatibility

The SDK maintains backward compatibility:

1. **Old YAML files** with `tier` field still work (field is ignored)
2. **Old License format** with `planType` still works
3. **Mix of old and new** works (SDK tries new format first, falls back to old)

You can migrate gradually:
- Update YAML files first → still works with old licenses
- Update licenses later → works with both old and new YAML

---

## Examples

### Example 1: Basic Feature

**Old:**
```yaml
- id: pdf_export
  tier: professional
  intercept:
    function: "GeneratePDF"
```

**New:**
```yaml
- id: pdf_export
  name: "PDF Export"
  description: "Export reports to PDF format"
  intercept:
    function: "GeneratePDF"
```

### Example 2: Feature with Quota

**Old:**
```yaml
- id: api_calls
  tier: enterprise
  quota:
    limit: 100000
    period: daily
  intercept:
    function: "HandleAPICall"
```

**New:**
```yaml
- id: api_calls
  name: "API Calls"
  intercept:
    function: "HandleAPICall"
  # Quota is in License, not here
```

License:
```json
{
  "features": {
    "api_calls": {
      "enabled": true,
      "quota": {"daily": 100000}
    }
  }
}
```

### Example 3: Feature with Fallback

**Old:**
```yaml
- id: advanced_search
  tier: professional
  intercept:
    function: "AdvancedSearch"
  on_deny:
    action: fallback
```

**New:**
```yaml
- id: advanced_search
  name: "Advanced Search"
  intercept:
    function: "AdvancedSearch"
  fallback:
    function: "BasicSearch"
  on_deny:
    action: fallback
```

---

## Benefits of New Model

1. **Clear Separation**
   - YAML: Technical mapping (feature ID → function)
   - License: Business rules (enabled/disabled, limits)

2. **Flexible Control**
   - Can enable/disable features per customer
   - Can set custom quotas per customer
   - No code changes needed

3. **Stable Interface**
   - Feature IDs are stable business interface
   - Function names can change (YAML mapping)
   - License format is independent

4. **Better Tiers**
   - "Professional" = License with specific features
   - Easy to create custom tiers
   - No tier hierarchy needed

---

## Troubleshooting

### Q: My old YAML has `tier` field, will it break?

**A**: No, it's ignored. The field is marked as deprecated but still accepted for backward compatibility.

### Q: My LCC server returns old license format, will it work?

**A**: Yes, the SDK tries the new format first, then falls back to the old tier-based check.

### Q: Can I mix old and new?

**A**: Yes:
- Old YAML + New License ✓
- New YAML + Old License ✓
- Mix of both ✓

### Q: When should I migrate?

**A**: You can migrate anytime. Recommended order:
1. Update YAML files (remove tier/quota)
2. Test with old licenses (should work)
3. Update LCC server to generate new license format
4. Test with new licenses

### Q: How do I know which model is being used?

**A**: Check the logs:
- Old: "checking tier requirement"
- New: "checking feature in license"

You can also check the License file structure.

---

## Support

If you encounter issues during migration:
1. Check backward compatibility notes above
2. Ensure both old and new formats work
3. Contact support with details

For questions about license generation:
- Contact your LCC server administrator
- Update LMF (License Manufacturing Facility) to generate new format
