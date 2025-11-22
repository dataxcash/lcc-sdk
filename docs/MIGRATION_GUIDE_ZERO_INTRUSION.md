# Migration Guide: Zero-Intrusion API

**Version**: 2.0  
**Date**: 2025-01-22  
**Status**: Complete

---

## Overview

This guide helps you migrate from the old invasive API (feature-level with `featureID` parameter) to the new **zero-intrusion API** (product-level without `featureID`).

### What Changed?

| Aspect | Old API (Deprecated) | New API (Zero-Intrusion) |
|--------|---------------------|-------------------------|
| **Scope** | Feature-level | Product-level |
| **Parameter** | Requires `featureID` | No `featureID` needed |
| **Helper Functions** | Not supported | Supported via `RegisterHelpers()` |
| **Code Generator** | Feature-based injection | Product-based injection |
| **Business Logic** | Invasive (must call SDK) | Zero-intrusion (auto-injected) |

---

## Quick Migration Examples

### 1. Consume API

**Old API (Deprecated):**
```go
allowed, remaining, reason, err := client.ConsumeDeprecated("feature-export", 10, nil)
if err != nil || !allowed {
    return fmt.Errorf("quota exceeded: %s", reason)
}
```

**New API (Zero-Intrusion):**
```go
allowed, remaining, err := client.Consume(10)
if err != nil || !allowed {
    return fmt.Errorf("quota exceeded")
}
```

**With Helper Function:**
```go
// Register helper once at startup
helpers := &client.HelperFunctions{
    QuotaConsumer: func(ctx context.Context, args ...interface{}) int {
        if len(args) > 0 {
            if batchSize, ok := args[0].(int); ok {
                return batchSize
            }
        }
        return 1
    },
    CapacityCounter: func() int {
        return database.CountActiveUsers()
    },
}
client.RegisterHelpers(helpers)

// Use in code (automatically calculates consumption from args)
allowed, remaining, err := client.ConsumeWithContext(ctx, batchSize)
```

---

### 2. CheckCapacity API

**Old API (Deprecated):**
```go
currentUsers := database.CountActiveUsers()
allowed, max, reason, err := client.CheckCapacityDeprecated("feature-users", currentUsers)
if err != nil || !allowed {
    return fmt.Errorf("capacity exceeded: %d/%d, %s", currentUsers, max, reason)
}
```

**New API (Zero-Intrusion with Manual Count):**
```go
currentUsers := database.CountActiveUsers()
allowed, max, err := client.CheckCapacity(currentUsers)
if err != nil || !allowed {
    return fmt.Errorf("capacity exceeded: %d/%d", currentUsers, max)
}
```

**New API (Zero-Intrusion with Helper):**
```go
// Register helper once at startup
helpers := &client.HelperFunctions{
    CapacityCounter: func() int {
        return database.CountActiveUsers()
    },
}
client.RegisterHelpers(helpers)

// Use in code (automatically calls CapacityCounter)
allowed, max, err := client.CheckCapacityWithHelper()
if err != nil || !allowed {
    return fmt.Errorf("capacity exceeded")
}
```

---

### 3. CheckTPS API

**Old API (Deprecated):**
```go
currentTPS := metrics.GetCurrentTPS()
allowed, max, reason, err := client.CheckTPSDeprecated("feature-api", currentTPS)
if err != nil || !allowed {
    return fmt.Errorf("TPS exceeded: %.2f/%.2f, %s", currentTPS, max, reason)
}
```

**New API (Zero-Intrusion):**
```go
// SDK automatically tracks TPS internally
allowed, max, err := client.CheckTPS()
if err != nil || !allowed {
    return fmt.Errorf("TPS exceeded: max=%.2f", max)
}
```

**With Custom TPS Provider:**
```go
// Register helper once at startup
helpers := &client.HelperFunctions{
    TPSProvider: func() float64 {
        return metrics.GetCurrentTPS()
    },
    CapacityCounter: func() int { return 0 }, // Stub if not using capacity
}
client.RegisterHelpers(helpers)

// SDK will use your TPSProvider
allowed, max, err := client.CheckTPS()
```

---

### 4. AcquireSlot API

**Old API (Deprecated):**
```go
release, allowed, reason, err := client.AcquireSlotDeprecated("feature-concurrent", nil)
if err != nil || !allowed {
    return fmt.Errorf("concurrency limit exceeded: %s", reason)
}
defer release()

// ... perform operation ...
```

**New API (Zero-Intrusion):**
```go
release, allowed, err := client.AcquireSlot()
if err != nil || !allowed {
    return fmt.Errorf("concurrency limit exceeded")
}
defer release()

// ... perform operation ...
```

---

## Configuration Migration

### Old YAML Format (Feature-Level Limits)

```yaml
sdk:
  lcc_url: "http://localhost:7086"
  product_id: "my-product"
  product_version: "1.0.0"

features:
  - id: "feature-export"
    name: "Data Export"
    quota:
      limit: 1000
      period: "daily"
    intercept:
      package: "export"
      function: "ExportData"
```

### New YAML Format (Product-Level Limits)

```yaml
sdk:
  lcc_url: "http://localhost:7086"
  product_id: "my-product"
  product_version: "1.0.0"
  
  # Product-level limits (applies to entire product)
  limits:
    quota:
      max: 1000
      window: "24h"
    max_tps: 100.0
    max_capacity: 500
    max_concurrency: 10
    
    # Optional: Helper function references for code generator
    consumer: "calculateBatchSize"
    capacity_counter: "countActiveUsers"

features:
  - id: "feature-export"
    name: "Data Export"
    # No limits here - just interception points
    intercept:
      package: "export"
      function: "ExportData"
```

---

## Step-by-Step Migration

### Step 1: Update Configuration

1. Move limits from `features[].quota` to `sdk.limits`
2. Define product-level limits: `quota`, `max_tps`, `max_capacity`, `max_concurrency`
3. Keep `features[]` array but remove limit fields

### Step 2: Update Code

#### Option A: Manual Migration (Simple Cases)

Replace old API calls with new ones:

```go
// Old
allowed, remaining, reason, err := client.ConsumeDeprecated(featureID, amount, nil)

// New
allowed, remaining, err := client.Consume(amount)
```

#### Option B: Code Generator (Recommended for Complex Cases)

1. Define helper functions:

```go
// helpers.go
package myapp

import "context"

func calculateBatchSize(ctx context.Context, args ...interface{}) int {
    if len(args) > 0 {
        if size, ok := args[0].(int); ok {
            return size
        }
    }
    return 1
}

func countActiveUsers() int {
    return database.CountActiveUsers()
}
```

2. Update YAML to reference helpers:

```yaml
sdk:
  limits:
    consumer: "calculateBatchSize"
    capacity_counter: "countActiveUsers"
```

3. Generate zero-intrusion wrappers:

```bash
lcc-codegen --mode=zero-intrusion --config=lcc-features.yaml --output=./generated
```

4. Register helpers at startup:

```go
helpers := &client.HelperFunctions{
    QuotaConsumer:   calculateBatchSize,
    CapacityCounter: countActiveUsers,
}
if err := lccClient.RegisterHelpers(helpers); err != nil {
    log.Fatalf("Failed to register helpers: %v", err)
}
```

### Step 3: Test

1. Run existing tests to verify backward compatibility
2. Test new API methods
3. Verify helper functions work correctly
4. Check that limits are enforced at product level

### Step 4: Remove Deprecated API Usage (Optional)

Once migration is complete and tested, you can remove deprecated API calls:

```go
// Remove these:
client.ConsumeDeprecated(...)
client.CheckCapacityDeprecated(...)
client.CheckTPSDeprecated(...)
client.AcquireSlotDeprecated(...)
```

---

## Benefits of Zero-Intrusion API

✅ **Cleaner Business Logic**: No license checks in business code  
✅ **Product-Level Control**: Unified limits across entire product  
✅ **Flexible Helpers**: Custom logic via helper functions  
✅ **Auto-Injection**: Code generator automatically adds checks  
✅ **Backward Compatible**: Old API still works during migration  

---

## Troubleshooting

### Error: "CapacityCounter is required"

**Cause**: Trying to use capacity limits without registering a helper.

**Solution**: 
```go
helpers := &client.HelperFunctions{
    CapacityCounter: func() int {
        return database.CountActiveUsers()
    },
}
client.RegisterHelpers(helpers)
```

Or provide a stub if not using capacity:
```go
helpers := &client.HelperFunctions{
    CapacityCounter: func() int { return 0 },
}
```

### Error: "no product limits defined"

**Cause**: Using zero-intrusion code generator without `sdk.limits` in YAML.

**Solution**: Add product limits to YAML:
```yaml
sdk:
  limits:
    quota:
      max: 1000
      window: "24h"
```

---

## Support

For questions or issues:
- Check examples in `examples/zero-intrusion/`
- Read helper functions guide: `docs/HELPER_FUNCTIONS_GUIDE.md`
- File an issue on GitHub

---

**Last Updated**: 2025-01-22
