# Zero-Intrusion API Example

This example demonstrates the **zero-intrusion API** of lcc-sdk, which enables product-level license enforcement without requiring `featureID` parameters in business code.

## Overview

The zero-intrusion API provides:

- **Product-level limits**: All limits apply at product level, not per-feature
- **No featureID parameters**: Clean API without feature-specific identifiers
- **Helper functions**: Custom callbacks for dynamic behavior
- **Internal TPS tracking**: Automatic TPS measurement without external dependencies
- **Backward compatibility**: Old API still works as deprecated methods

## Files

- `lcc-features.yaml` - Configuration with product-level limits
- `main.go` - Example code showing all zero-intrusion API methods

## Product-Level Limits

The configuration defines limits at the product level:

```yaml
sdk:
  limits:
    quota:
      max: 1000
      window: "24h"
    max_tps: 100.0
    max_capacity: 500
    max_concurrency: 10
```

These limits apply to the **entire product**, not individual features.

## Helper Functions

Helper functions provide custom logic without modifying business code:

### QuotaConsumer (Optional)

Calculate consumption amount based on function arguments:

```go
QuotaConsumer: func(ctx context.Context, args ...interface{}) int {
    if len(args) > 0 {
        if batchSize, ok := args[0].(int); ok {
            return batchSize
        }
    }
    return 1
}
```

### TPSProvider (Optional)

Provide current TPS measurement:

```go
TPSProvider: func() float64 {
    return getCurrentTPS()
}
```

If not provided, SDK tracks TPS internally.

### CapacityCounter (Required)

Count current resource usage:

```go
CapacityCounter: func() int {
    return countActiveUsers()
}
```

Required if using capacity limits.

## API Methods

### 1. Consume

Simple quota consumption:

```go
allowed, remaining, err := client.Consume(10)
```

With helper function:

```go
allowed, remaining, err := client.ConsumeWithContext(ctx, batchSize)
```

### 2. CheckTPS

Check transactions per second:

```go
allowed, maxTPS, err := client.CheckTPS()
```

Uses TPSProvider helper if registered, otherwise uses internal tracking.

### 3. CheckCapacity

Check capacity limits:

```go
// Manual count
allowed, max, err := client.CheckCapacity(currentUsers)

// Using helper
allowed, max, err := client.CheckCapacityWithHelper()
```

### 4. AcquireSlot

Concurrency control:

```go
release, allowed, err := client.AcquireSlot()
if err != nil || !allowed {
    return fmt.Errorf("concurrency limit exceeded")
}
defer release()

// ... perform operation ...
```

## Running the Example

1. Start LCC server:
   ```bash
   # Make sure LCC server is running on localhost:7086
   ```

2. Run the example:
   ```bash
   go run main.go
   ```

## Expected Output

```
✅ Example 1: Consumed 1 unit, 999 remaining
✅ Example 2: Consumed 10 units, 989 remaining
✅ Example 3: TPS check passed (max=100.00)
✅ Example 4: Capacity check passed (max=500)
✅ Example 5: Slot acquired, performing operation...
All examples completed successfully!
```

## Comparison with Old API

### Old API (Deprecated)

```go
// Feature-level, requires featureID
allowed, remaining, reason, err := client.ConsumeDeprecated("feature-id", 10, nil)
allowed, max, reason, err := client.CheckTPSDeprecated("feature-id", currentTPS)
allowed, max, reason, err := client.CheckCapacityDeprecated("feature-id", currentUsed)
release, allowed, reason, err := client.AcquireSlotDeprecated("feature-id", nil)
```

### New API (Zero-Intrusion)

```go
// Product-level, no featureID needed
allowed, remaining, err := client.Consume(10)
allowed, max, err := client.CheckTPS()
allowed, max, err := client.CheckCapacity(currentUsed)
release, allowed, err := client.AcquireSlot()
```

## Benefits

✅ **Cleaner Code**: No license checks in business logic  
✅ **Product-Level Control**: Unified limits across entire product  
✅ **Flexible Helpers**: Custom logic via helper functions  
✅ **Auto-Injection**: Code generator automatically adds checks  
✅ **Backward Compatible**: Old API still works during migration  

## Next Steps

1. Read the [Migration Guide](../../docs/MIGRATION_GUIDE_ZERO_INTRUSION.md)
2. Review [Helper Functions Guide](../../docs/HELPER_FUNCTIONS_GUIDE.md)
3. Check [Refactoring Completion Report](../../docs/REFACTORING_COMPLETION_ZERO_INTRUSION.md)

## See Also

- [lcc-demo-app](https://github.com/yourorg/lcc-demo-app) - Complete demo application
- [SDK Documentation](../../README.md)
- [API Reference](../../docs/api-reference.md)
