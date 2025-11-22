# Helper Functions Guide

**Version**: 2.0  
**Date**: 2025-01-22

---

## Overview

Helper functions are the key to **zero-intrusion** license enforcement in lcc-sdk. They allow you to provide custom logic for calculating quota consumption, measuring TPS, and counting capacity—all without modifying your business code.

### What Are Helper Functions?

Helper functions are callback functions you register with the LCC client that:

1. **Calculate dynamic values** (e.g., quota consumption based on batch size)
2. **Provide current metrics** (e.g., TPS from your monitoring system)
3. **Count resources** (e.g., active users in your database)

The SDK calls these functions automatically when checking limits, enabling zero-intrusion enforcement.

---

## Helper Function Types

### 1. QuotaConsumer (Optional)

**Purpose**: Calculate how many quota units to consume based on function arguments.

**Signature**:
```go
func(ctx context.Context, args ...interface{}) int
```

**When to Use**:
- Variable consumption amounts (e.g., batch operations)
- Consumption based on data size
- Complex quota calculation logic

**Default Behavior**: If not provided, SDK consumes 1 unit per call.

**Example - Simple**:
```go
QuotaConsumer: func(ctx context.Context, args ...interface{}) int {
    return 1 // Always consume 1 unit
}
```

**Example - Batch Size**:
```go
QuotaConsumer: func(ctx context.Context, args ...interface{}) int {
    if len(args) > 0 {
        if batchSize, ok := args[0].(int); ok {
            return batchSize
        }
    }
    return 1 // Fallback to 1
}
```

**Example - Data Size**:
```go
QuotaConsumer: func(ctx context.Context, args ...interface{}) int {
    if len(args) > 0 {
        if data, ok := args[0].([]byte); ok {
            // Consume 1 unit per KB
            return (len(data) + 1023) / 1024
        }
    }
    return 1
}
```

---

### 2. TPSProvider (Optional)

**Purpose**: Provide current transactions per second measurement.

**Signature**:
```go
func() float64
```

**When to Use**:
- You have existing TPS monitoring
- Need precise TPS measurements
- Want to use custom TPS calculation logic

**Default Behavior**: If not provided, SDK tracks TPS internally using a sliding window.

**Example - From Metrics System**:
```go
TPSProvider: func() float64 {
    return myMetrics.GetCurrentTPS()
}
```

**Example - From Prometheus**:
```go
TPSProvider: func() float64 {
    result, err := prometheus.Query("rate(http_requests_total[1m])")
    if err != nil {
        return 0
    }
    return result.Value
}
```

**Example - Average Over Window**:
```go
TPSProvider: func() float64 {
    now := time.Now()
    window := 10 * time.Second
    count := requestCounter.CountSince(now.Add(-window))
    return float64(count) / window.Seconds()
}
```

---

### 3. CapacityCounter (Required)

**Purpose**: Count current usage of persistent resources.

**Signature**:
```go
func() int
```

**When to Use**:
- Capacity limits are configured
- Need to count active users, connections, or other resources
- Must provide if using `max_capacity` in config

**Default Behavior**: No default—MUST be provided if using capacity limits.

**Example - Active Users**:
```go
CapacityCounter: func() int {
    return database.CountActiveUsers()
}
```

**Example - Open Connections**:
```go
CapacityCounter: func() int {
    return connectionPool.ActiveConnections()
}
```

**Example - Created Items**:
```go
CapacityCounter: func() int {
    count, _ := redis.Get("item_count").Int()
    return count
}
```

**Example - Stub (Not Using Capacity)**:
```go
CapacityCounter: func() int {
    return 0 // Stub if not using capacity limits
}
```

---

## Registering Helper Functions

### Basic Registration

```go
import "github.com/yourorg/lcc-sdk/pkg/client"

// Create helpers
helpers := &client.HelperFunctions{
    QuotaConsumer: func(ctx context.Context, args ...interface{}) int {
        return 1
    },
    CapacityCounter: func() int {
        return database.CountActiveUsers()
    },
}

// Register with client
if err := lccClient.RegisterHelpers(helpers); err != nil {
    log.Fatalf("Failed to register helpers: %v", err)
}
```

### Registration with Validation

```go
helpers := &client.HelperFunctions{
    QuotaConsumer: myQuotaConsumer,
    TPSProvider:   myTPSProvider,
    CapacityCounter: myCapacityCounter,
}

// Validate before registering
if err := helpers.Validate(); err != nil {
    log.Fatalf("Helper validation failed: %v", err)
}

// Register
if err := lccClient.RegisterHelpers(helpers); err != nil {
    log.Fatalf("Failed to register helpers: %v", err)
}
```

---

## Complete Examples

### Example 1: Simple Application

```go
package main

import (
    "context"
    "log"
    
    "github.com/yourorg/lcc-sdk/pkg/client"
    "github.com/yourorg/lcc-sdk/pkg/config"
)

var database *Database

func main() {
    // Load config
    cfg := &config.SDKConfig{
        LCCURL:         "http://localhost:7086",
        ProductID:      "my-app",
        ProductVersion: "1.0.0",
    }
    
    // Create LCC client
    lccClient, err := client.NewClient(cfg)
    if err != nil {
        log.Fatal(err)
    }
    defer lccClient.Close()
    
    // Register helpers
    helpers := &client.HelperFunctions{
        CapacityCounter: func() int {
            return database.CountActiveUsers()
        },
    }
    
    if err := lccClient.RegisterHelpers(helpers); err != nil {
        log.Fatal(err)
    }
    
    // Register with LCC server
    if err := lccClient.Register(); err != nil {
        log.Fatal(err)
    }
    
    // Use zero-intrusion API
    if err := exportData(lccClient); err != nil {
        log.Fatal(err)
    }
}

func exportData(lccClient *client.Client) error {
    // Check capacity before exporting
    allowed, max, err := lccClient.CheckCapacityWithHelper()
    if err != nil || !allowed {
        return fmt.Errorf("capacity exceeded: max=%d", max)
    }
    
    // Consume quota
    allowed, remaining, err := lccClient.Consume(1)
    if err != nil || !allowed {
        return fmt.Errorf("quota exceeded: remaining=%d", remaining)
    }
    
    // Perform export
    // ...
    
    return nil
}
```

---

### Example 2: Batch Processing

```go
package main

import (
    "context"
    
    "github.com/yourorg/lcc-sdk/pkg/client"
)

func setupBatchHelpers(lccClient *client.Client) error {
    helpers := &client.HelperFunctions{
        // Consume quota based on batch size
        QuotaConsumer: func(ctx context.Context, args ...interface{}) int {
            if len(args) > 0 {
                if items, ok := args[0].([]interface{}); ok {
                    return len(items)
                }
            }
            return 1
        },
        
        // Stub for capacity (not using it)
        CapacityCounter: func() int {
            return 0
        },
    }
    
    return lccClient.RegisterHelpers(helpers)
}

func processBatch(lccClient *client.Client, items []interface{}) error {
    ctx := context.Background()
    
    // Automatically consumes len(items) quota units
    allowed, remaining, err := lccClient.ConsumeWithContext(ctx, items)
    if err != nil || !allowed {
        return fmt.Errorf("quota exceeded: remaining=%d", remaining)
    }
    
    // Process items
    for _, item := range items {
        // ...
    }
    
    return nil
}
```

---

### Example 3: API Server with TPS Monitoring

```go
package main

import (
    "sync/atomic"
    "time"
    
    "github.com/yourorg/lcc-sdk/pkg/client"
)

type TPSMonitor struct {
    requestCount int64
    lastReset    time.Time
}

func (m *TPSMonitor) RecordRequest() {
    atomic.AddInt64(&m.requestCount, 1)
}

func (m *TPSMonitor) GetCurrentTPS() float64 {
    count := atomic.LoadInt64(&m.requestCount)
    elapsed := time.Since(m.lastReset).Seconds()
    if elapsed > 0 {
        return float64(count) / elapsed
    }
    return 0
}

func (m *TPSMonitor) Reset() {
    atomic.StoreInt64(&m.requestCount, 0)
    m.lastReset = time.Now()
}

var tpsMonitor = &TPSMonitor{lastReset: time.Now()}

func setupAPIHelpers(lccClient *client.Client) error {
    // Reset TPS counter every second
    go func() {
        ticker := time.NewTicker(time.Second)
        defer ticker.Stop()
        for range ticker.C {
            tpsMonitor.Reset()
        }
    }()
    
    helpers := &client.HelperFunctions{
        TPSProvider: func() float64 {
            return tpsMonitor.GetCurrentTPS()
        },
        CapacityCounter: func() int {
            return database.CountActiveConnections()
        },
    }
    
    return lccClient.RegisterHelpers(helpers)
}

func handleAPIRequest(lccClient *client.Client) error {
    // Record request for TPS tracking
    tpsMonitor.RecordRequest()
    
    // Check TPS limit
    allowed, maxTPS, err := lccClient.CheckTPS()
    if err != nil || !allowed {
        return fmt.Errorf("TPS exceeded: max=%.2f", maxTPS)
    }
    
    // Process request
    // ...
    
    return nil
}
```

---

## Best Practices

### 1. Performance

✅ **DO**: Keep helper functions fast  
✅ **DO**: Cache expensive calculations  
✅ **DO**: Use atomic operations for counters  

❌ **DON'T**: Make database queries in every call  
❌ **DON'T**: Block for long periods  
❌ **DON'T**: Use locks unnecessarily  

### 2. Error Handling

✅ **DO**: Return sensible defaults on error  
✅ **DO**: Log errors internally  
✅ **DO**: Handle nil/invalid inputs gracefully  

❌ **DON'T**: Panic in helper functions  
❌ **DON'T**: Return negative values  
❌ **DON'T**: Ignore all errors silently  

### 3. Thread Safety

✅ **DO**: Use atomic operations for shared state  
✅ **DO**: Use read locks for read-only operations  
✅ **DO**: Make helpers concurrent-safe  

❌ **DON'T**: Use non-threadsafe data structures  
❌ **DON'T**: Modify shared state without locking  

---

## Testing Helper Functions

### Unit Testing

```go
func TestQuotaConsumer(t *testing.T) {
    consumer := func(ctx context.Context, args ...interface{}) int {
        if len(args) > 0 {
            if size, ok := args[0].(int); ok {
                return size
            }
        }
        return 1
    }
    
    tests := []struct {
        args     []interface{}
        expected int
    }{
        {[]interface{}{10}, 10},
        {[]interface{}{"invalid"}, 1},
        {[]interface{}{}, 1},
    }
    
    for _, tt := range tests {
        result := consumer(context.Background(), tt.args...)
        if result != tt.expected {
            t.Errorf("expected %d, got %d", tt.expected, result)
        }
    }
}
```

### Integration Testing

```go
func TestHelpersWithClient(t *testing.T) {
    cfg := &config.SDKConfig{
        LCCURL:    "http://localhost:7086",
        ProductID: "test-app",
        ProductVersion: "1.0.0",
    }
    
    client, err := client.NewClient(cfg)
    require.NoError(t, err)
    defer client.Close()
    
    helpers := &client.HelperFunctions{
        QuotaConsumer: func(ctx context.Context, args ...interface{}) int {
            return 5
        },
        CapacityCounter: func() int {
            return 10
        },
    }
    
    err = client.RegisterHelpers(helpers)
    require.NoError(t, err)
    
    // Test ConsumeWithContext
    allowed, remaining, err := client.ConsumeWithContext(context.Background())
    require.NoError(t, err)
    assert.True(t, allowed)
    
    // Test CheckCapacityWithHelper
    allowed, max, err := client.CheckCapacityWithHelper()
    require.NoError(t, err)
    assert.True(t, allowed)
}
```

---

## Troubleshooting

### Helper Not Being Called

**Check**:
1. Helper was registered with `RegisterHelpers()`
2. Using correct API method (`ConsumeWithContext` vs `Consume`)
3. No errors during registration

### Helper Returns Wrong Value

**Check**:
1. Arguments are being passed correctly
2. Type assertions succeed
3. Fallback logic is correct

### Performance Issues

**Check**:
1. Helper function is not too slow
2. No blocking operations
3. Consider caching results

---

## See Also

- Migration Guide: `MIGRATION_GUIDE_ZERO_INTRUSION.md`
- API Reference: `pkg/client/helpers.go`
- Examples: `examples/zero-intrusion/`

---

**Last Updated**: 2025-01-22
