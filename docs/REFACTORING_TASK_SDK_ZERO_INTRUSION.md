# LCC SDK Refactoring Task: Zero-Intrusion Implementation

**Date**: 2025-01-22  
**Priority**: High  
**Status**: Ready to Start  
**Objective**: Implement true zero-intrusion API in lcc-sdk based on demo app design

---

## 📋 Context

### What Has Been Done
✅ **lcc-demo-app** has been refactored to showcase the **ideal zero-intrusion design**:
- Product-level limits (not feature-level)
- Zero-intrusion code examples
- Helper function system documentation
- YAML configuration format defined

### What Needs to Be Done
⚠️ **lcc-sdk** still uses the **old invasive API design**:
- API methods require `featureID` parameter (invasive)
- No helper function registration system
- No compiler/code generator
- Feature-level instead of product-level

### Reference Documents
- Demo App Refactoring Report: `/home/fila/jqdDev_2025/lcc-demo-app/docs/REFACTORING_COMPLETION_ZERO_INTRUSION.md`
- Design Comparison: `/home/fila/jqdDev_2025/lcc-demo-app/docs/ZERO_INTRUSION_COMPARISON.md`
- Original Task Spec: `/home/fila/jqdDev_2025/lcc-demo-app/docs/REFACTORING_TASK_ZERO_INTRUSION.md`

---

## 🎯 Goals

### Core Objectives
1. ✅ **Product-Level API**: Remove `featureID` from all limit check APIs
2. ✅ **Helper Function System**: Implement registration and execution
3. ✅ **Zero-Intrusion**: Business logic stays clean (no license code)
4. ✅ **Backward Compatibility**: Support old API during transition

### Success Criteria
- [ ] Client API uses product-level limits (no featureID parameter)
- [ ] Helper functions can be registered and called
- [ ] Code generator can inject license checks
- [ ] All existing tests pass
- [ ] New tests for helper function system
- [ ] Documentation updated

---

## 📦 Architecture Overview

### Current Architecture (Invasive)
```go
// pkg/client/client.go - OLD API
client.Consume(featureID string, amount int, meta map[string]any)
client.CheckTPS(featureID string, currentTPS float64)
client.CheckCapacity(featureID string, currentUsed int)
client.AcquireSlot(featureID string, meta map[string]any)
```

**Problems:**
- ❌ featureID required in every call
- ❌ Business logic must call license APIs manually
- ❌ No separation of concerns
- ❌ Feature-level instead of product-level

### Target Architecture (Zero-Intrusion)
```go
// pkg/client/client.go - NEW API
client.Consume(amount int) (bool, int, error)
client.CheckTPS() (bool, float64, error)
client.CheckCapacity(currentUsed int) (bool, int, error)
client.AcquireSlot() (ReleaseFunc, bool, error)

// pkg/client/helpers.go - Helper Function System
type HelperFunctions struct {
    QuotaConsumer   func(ctx context.Context, args ...interface{}) int
    TPSProvider     func() float64
    CapacityCounter func() int
}

client.RegisterHelpers(helpers)

// pkg/codegen/generator.go - Code Generator
generator.Generate(yamlConfig, outputDir)
```

**Benefits:**
- ✅ No featureID needed (product-level)
- ✅ Helper functions provide flexibility
- ✅ Code generator auto-injects checks
- ✅ Clean business logic

---

## 🗓️ Implementation Plan

### Phase 1: API Refactoring (Core)
**Goal**: Refactor Client API to be product-level

#### Task 1.1: Add Helper Function Types
**File**: `pkg/client/helpers.go` (new file)

```go
package client

import "context"

// HelperFunctions contains optional/required helper functions
// for customizing limit enforcement behavior
type HelperFunctions struct {
    // QuotaConsumer (Optional): Calculate custom consumption amount
    // If not provided, defaults to consuming 1 unit per call
    // Args: function parameters from intercepted call
    QuotaConsumer func(ctx context.Context, args ...interface{}) int
    
    // TPSProvider (Optional): Provide current TPS measurement
    // If not provided, SDK auto-tracks TPS internally
    TPSProvider func() float64
    
    // CapacityCounter (Required): Count current resource usage
    // MUST be provided for capacity limits to work
    // Returns: current count of persistent resources
    CapacityCounter func() int
}

// Default implementations
func defaultQuotaConsumer(ctx context.Context, args ...interface{}) int {
    return 1 // Default: consume 1 unit
}

func defaultTPSProvider() float64 {
    // SDK internal TPS tracking
    return getCurrentTPSFromSDK()
}
```

**Checklist:**
- [ ] Create `pkg/client/helpers.go`
- [ ] Define `HelperFunctions` struct
- [ ] Add default implementations
- [ ] Add validation functions
- [ ] Add documentation comments

---

#### Task 1.2: Add Helper Registration to Client
**File**: `pkg/client/client.go`

```go
type Client struct {
    // ... existing fields ...
    
    // New fields for zero-intrusion
    helpers *HelperFunctions
    tpsTracker *tpsTracker // Internal TPS tracking
}

// RegisterHelpers registers helper functions for custom limit behavior
func (c *Client) RegisterHelpers(helpers *HelperFunctions) error {
    if helpers == nil {
        return fmt.Errorf("helpers cannot be nil")
    }
    
    // Validate required helpers
    if helpers.CapacityCounter == nil {
        return fmt.Errorf("CapacityCounter is required")
    }
    
    // Set defaults for optional helpers
    if helpers.QuotaConsumer == nil {
        helpers.QuotaConsumer = defaultQuotaConsumer
    }
    if helpers.TPSProvider == nil {
        helpers.TPSProvider = c.getInternalTPS
    }
    
    c.helpers = helpers
    return nil
}
```

**Checklist:**
- [ ] Add `helpers` field to Client struct
- [ ] Implement `RegisterHelpers()` method
- [ ] Add validation logic
- [ ] Add default helper assignments
- [ ] Update `NewClient()` to initialize helpers

---

#### Task 1.3: Refactor Consume() API
**File**: `pkg/client/client.go`

```go
// OLD API (keep for backward compatibility)
func (c *Client) ConsumeDeprecated(featureID string, amount int, meta map[string]any) (bool, int, string, error) {
    // Delegate to old implementation
}

// NEW API (product-level, zero-intrusion)
// Consume consumes quota from the product-level quota pool
// Amount is determined by QuotaConsumer helper or defaults to 1
func (c *Client) Consume(amount int) (bool, int, error) {
    // Check product-level quota
    status, err := c.checkProductLimits()
    if err != nil {
        return false, 0, err
    }
    
    if !status.Enabled {
        return false, status.Quota.Remaining, fmt.Errorf("quota exceeded")
    }
    
    // Report usage
    if err := c.reportProductUsage(amount); err != nil {
        return false, 0, err
    }
    
    remaining := status.Quota.Remaining - amount
    if remaining < 0 {
        remaining = 0
    }
    
    return true, remaining, nil
}

// ConsumeWithContext uses registered QuotaConsumer helper
func (c *Client) ConsumeWithContext(ctx context.Context, args ...interface{}) (bool, int, error) {
    if c.helpers == nil || c.helpers.QuotaConsumer == nil {
        return false, 0, fmt.Errorf("QuotaConsumer helper not registered")
    }
    
    amount := c.helpers.QuotaConsumer(ctx, args...)
    return c.Consume(amount)
}
```

**Checklist:**
- [ ] Rename old `Consume()` to `ConsumeDeprecated()`
- [ ] Implement new product-level `Consume(amount)`
- [ ] Add `ConsumeWithContext()` for helper integration
- [ ] Implement `checkProductLimits()` helper
- [ ] Implement `reportProductUsage()` helper
- [ ] Add tests

---

#### Task 1.4: Refactor CheckTPS() API
**File**: `pkg/client/client.go`

```go
// OLD API
func (c *Client) CheckTPSDeprecated(featureID string, currentTPS float64) (bool, float64, string, error) {
    // Keep for backward compatibility
}

// NEW API (product-level)
// CheckTPS checks current TPS against product-level limit
// Uses TPSProvider helper or SDK internal tracking
func (c *Client) CheckTPS() (bool, float64, error) {
    // Get current TPS from helper or internal tracker
    currentTPS := c.getCurrentTPS()
    
    // Check against product limit
    status, err := c.checkProductLimits()
    if err != nil {
        return false, 0, err
    }
    
    maxTPS := status.MaxTPS
    if maxTPS <= 0 {
        return true, 0, nil // No TPS limit configured
    }
    
    if currentTPS > maxTPS {
        return false, maxTPS, fmt.Errorf("TPS exceeded: %.2f > %.2f", currentTPS, maxTPS)
    }
    
    return true, maxTPS, nil
}

// getCurrentTPS gets TPS from helper or internal tracker
func (c *Client) getCurrentTPS() float64 {
    if c.helpers != nil && c.helpers.TPSProvider != nil {
        return c.helpers.TPSProvider()
    }
    return c.tpsTracker.getCurrentRate()
}
```

**Checklist:**
- [ ] Rename old `CheckTPS()` to `CheckTPSDeprecated()`
- [ ] Implement new product-level `CheckTPS()`
- [ ] Add `getCurrentTPS()` helper method
- [ ] Implement internal TPS tracker
- [ ] Add tests

---

#### Task 1.5: Refactor CheckCapacity() API
**File**: `pkg/client/client.go`

```go
// OLD API
func (c *Client) CheckCapacityDeprecated(featureID string, currentUsed int) (bool, int, string, error) {
    // Keep for backward compatibility
}

// NEW API (product-level)
// CheckCapacity checks current usage against product-level capacity limit
// Uses CapacityCounter helper (REQUIRED)
func (c *Client) CheckCapacity(currentUsed int) (bool, int, error) {
    status, err := c.checkProductLimits()
    if err != nil {
        return false, 0, err
    }
    
    maxCapacity := status.MaxCapacity
    if maxCapacity <= 0 {
        return false, 0, fmt.Errorf("no capacity limit configured")
    }
    
    if currentUsed >= maxCapacity {
        return false, maxCapacity, fmt.Errorf("capacity exceeded: %d >= %d", currentUsed, maxCapacity)
    }
    
    return true, maxCapacity, nil
}

// CheckCapacityWithHelper uses registered CapacityCounter helper
func (c *Client) CheckCapacityWithHelper() (bool, int, error) {
    if c.helpers == nil || c.helpers.CapacityCounter == nil {
        return false, 0, fmt.Errorf("CapacityCounter helper not registered (required)")
    }
    
    currentUsed := c.helpers.CapacityCounter()
    return c.CheckCapacity(currentUsed)
}
```

**Checklist:**
- [ ] Rename old `CheckCapacity()` to `CheckCapacityDeprecated()`
- [ ] Implement new product-level `CheckCapacity()`
- [ ] Add `CheckCapacityWithHelper()` for convenience
- [ ] Add validation for required helper
- [ ] Add tests

---

#### Task 1.6: Refactor AcquireSlot() API
**File**: `pkg/client/client.go`

```go
// OLD API
func (c *Client) AcquireSlotDeprecated(featureID string, meta map[string]any) (func(), bool, string, error) {
    // Keep for backward compatibility
}

// NEW API (product-level)
// AcquireSlot acquires a slot from product-level concurrency pool
// No helper needed - SDK manages automatically
func (c *Client) AcquireSlot() (ReleaseFunc, bool, error) {
    status, err := c.checkProductLimits()
    if err != nil {
        return func() {}, false, err
    }
    
    maxConcurrency := status.MaxConcurrency
    if maxConcurrency <= 0 {
        return func() {}, false, fmt.Errorf("no concurrency limit configured")
    }
    
    // Acquire from product-level pool
    c.mu.Lock()
    defer c.mu.Unlock()
    
    key := c.instanceID + "::product"
    current := concurrencyState[key]
    
    if current >= maxConcurrency {
        return func() {}, false, fmt.Errorf("concurrency exceeded: %d >= %d", current, maxConcurrency)
    }
    
    concurrencyState[key] = current + 1
    
    release := func() {
        c.mu.Lock()
        defer c.mu.Unlock()
        cur := concurrencyState[key]
        if cur <= 1 {
            delete(concurrencyState, key)
        } else {
            concurrencyState[key] = cur - 1
        }
    }
    
    return release, true, nil
}

type ReleaseFunc func()
```

**Checklist:**
- [ ] Rename old `AcquireSlot()` to `AcquireSlotDeprecated()`
- [ ] Implement new product-level `AcquireSlot()`
- [ ] Define `ReleaseFunc` type
- [ ] Update concurrency tracking to use product-level key
- [ ] Add tests

---

### Phase 2: Internal TPS Tracking
**Goal**: Implement SDK internal TPS measurement

#### Task 2.1: Create TPS Tracker
**File**: `pkg/client/tps_tracker.go` (new file)

```go
package client

import (
    "sync"
    "time"
)

// tpsTracker tracks transactions per second internally
type tpsTracker struct {
    mu        sync.RWMutex
    requests  []time.Time
    window    time.Duration
}

func newTPSTracker() *tpsTracker {
    return &tpsTracker{
        requests: make([]time.Time, 0, 100),
        window:   time.Second,
    }
}

// RecordRequest records a new request
func (t *tpsTracker) RecordRequest() {
    t.mu.Lock()
    defer t.mu.Unlock()
    
    now := time.Now()
    t.requests = append(t.requests, now)
    
    // Clean old requests
    cutoff := now.Add(-t.window)
    validIdx := 0
    for i, req := range t.requests {
        if req.After(cutoff) {
            validIdx = i
            break
        }
    }
    t.requests = t.requests[validIdx:]
}

// getCurrentRate returns current TPS
func (t *tpsTracker) getCurrentRate() float64 {
    t.mu.RLock()
    defer t.mu.RUnlock()
    
    now := time.Now()
    cutoff := now.Add(-t.window)
    
    count := 0
    for _, req := range t.requests {
        if req.After(cutoff) {
            count++
        }
    }
    
    return float64(count)
}
```

**Checklist:**
- [ ] Create `pkg/client/tps_tracker.go`
- [ ] Implement `tpsTracker` struct
- [ ] Add `RecordRequest()` method
- [ ] Add `getCurrentRate()` method
- [ ] Add tests for TPS calculation
- [ ] Integrate with Client

---

### Phase 3: Code Generator
**Goal**: Implement compiler/code generator for auto-injection

#### Task 3.1: Enhance YAML Parser
**File**: `pkg/config/types.go`

```go
// Add product-level limits to config
type SDKConfig struct {
    ProductID      string `yaml:"product_id"`
    ProductVersion string `yaml:"product_version"`
    LCCURL         string `yaml:"lcc_url"`
    
    // Product-level limits (NEW)
    Limits *ProductLimits `yaml:"limits"`
    
    Features []FeatureConfig `yaml:"features"`
    // ... other fields
}

type ProductLimits struct {
    Quota           *QuotaConfig `yaml:"quota"`
    MaxTPS          float64      `yaml:"max_tps"`
    MaxCapacity     int          `yaml:"max_capacity"`
    MaxConcurrency  int          `yaml:"max_concurrency"`
    
    // Helper function references
    QuotaConsumer   string `yaml:"consumer"`       // Optional
    TPSProvider     string `yaml:"tps_provider"`   // Optional
    CapacityCounter string `yaml:"capacity_counter"` // Required
}

type QuotaConfig struct {
    Max    int    `yaml:"max"`
    Window string `yaml:"window"`
}

// Remove limits from FeatureConfig (features are just interception points)
type FeatureConfig struct {
    ID        string            `yaml:"id"`
    Name      string            `yaml:"name"`
    Intercept InterceptConfig   `yaml:"intercept"`
    OnDeny    OnDenyConfig      `yaml:"on_deny"`
}
```

**Checklist:**
- [ ] Add `ProductLimits` struct
- [ ] Add `QuotaConfig` struct
- [ ] Add helper function reference fields
- [ ] Remove limit fields from `FeatureConfig`
- [ ] Update parser tests

---

#### Task 3.2: Implement Code Generator
**File**: `pkg/codegen/generator.go`

```go
package codegen

import (
    "fmt"
    "go/ast"
    "go/parser"
    "go/token"
    "text/template"
    
    "github.com/yourorg/lcc-sdk/pkg/config"
)

// Generator generates wrapper code with license checks
type Generator struct {
    config *config.SDKConfig
    fset   *token.FileSet
}

func NewGenerator(cfg *config.SDKConfig) *Generator {
    return &Generator{
        config: cfg,
        fset:   token.NewFileSet(),
    }
}

// Generate generates wrapper code for all intercepted functions
func (g *Generator) Generate(srcDir, outputDir string) error {
    // 1. Parse source files
    // 2. Find functions to intercept
    // 3. Generate wrapper functions
    // 4. Write generated code
    
    for _, feature := range g.config.Features {
        if err := g.generateFeatureWrapper(feature, outputDir); err != nil {
            return err
        }
    }
    
    return nil
}

// generateFeatureWrapper generates a wrapper for one feature
func (g *Generator) generateFeatureWrapper(feature config.FeatureConfig, outputDir string) error {
    tmpl, err := template.New("wrapper").Parse(wrapperTemplate)
    if err != nil {
        return err
    }
    
    data := struct {
        Feature config.FeatureConfig
        Limits  *config.ProductLimits
    }{
        Feature: feature,
        Limits:  g.config.Limits,
    }
    
    // Generate wrapper code
    var buf bytes.Buffer
    if err := tmpl.Execute(&buf, data); err != nil {
        return err
    }
    
    // Write to file
    outputPath := filepath.Join(outputDir, feature.ID+"_wrapper.go")
    return os.WriteFile(outputPath, buf.Bytes(), 0644)
}

const wrapperTemplate = `
// Auto-generated by lcc-sdk code generator
// DO NOT EDIT

package {{.Feature.Intercept.Package}}

import (
    "context"
    "fmt"
)

// Original function signature (placeholder - will be detected from source)
// func {{.Feature.Intercept.Function}}(args...) (returns...)

// Generated wrapper with license checks
func {{.Feature.Intercept.Function}}__lcc_wrapper(args...) (returns...) {
    {{if .Limits.MaxConcurrency}}
    // Auto-injected: Concurrency control
    release, allowed, err := __lcc_client.AcquireSlot()
    if err != nil || !allowed {
        return {{.Feature.OnDeny.GetErrorReturn}}
    }
    defer release()
    {{end}}
    
    {{if .Limits.Quota}}
    // Auto-injected: Quota consumption
    {{if .Limits.QuotaConsumer}}
    amount := {{.Limits.QuotaConsumer}}(context.Background(), args...)
    {{else}}
    amount := 1
    {{end}}
    allowed, remaining, err := __lcc_client.Consume(amount)
    if err != nil || !allowed {
        return {{.Feature.OnDeny.GetErrorReturn}}
    }
    {{end}}
    
    {{if .Limits.MaxTPS}}
    // Auto-injected: TPS check
    allowed, maxTPS, err := __lcc_client.CheckTPS()
    if err != nil || !allowed {
        return {{.Feature.OnDeny.GetErrorReturn}}
    }
    {{end}}
    
    {{if .Limits.MaxCapacity}}
    // Auto-injected: Capacity check
    allowed, maxCap, err := __lcc_client.CheckCapacityWithHelper()
    if err != nil || !allowed {
        return {{.Feature.OnDeny.GetErrorReturn}}
    }
    {{end}}
    
    // Call original business logic
    return {{.Feature.Intercept.Function}}__original(args...)
}
`
```

**Checklist:**
- [ ] Implement `Generator` struct
- [ ] Implement `Generate()` method
- [ ] Create wrapper templates
- [ ] Add function signature detection
- [ ] Add tests for code generation
- [ ] Create CLI tool for generator

---

### Phase 4: Testing & Documentation
**Goal**: Comprehensive testing and documentation

#### Task 4.1: Unit Tests
- [ ] Test helper function registration
- [ ] Test product-level API methods
- [ ] Test TPS tracker accuracy
- [ ] Test code generator output
- [ ] Test backward compatibility

#### Task 4.2: Integration Tests
- [ ] Test end-to-end with generated code
- [ ] Test with real LCC server
- [ ] Test helper function execution
- [ ] Test error scenarios

#### Task 4.3: Documentation
- [ ] Update README.md
- [ ] Create MIGRATION_GUIDE.md (old API → new API)
- [ ] Create HELPER_FUNCTIONS_GUIDE.md
- [ ] Update API documentation
- [ ] Add code examples

---

## 📁 File Structure

```
lcc-sdk/
├── pkg/
│   ├── client/
│   │   ├── client.go           # Modified: new product-level API
│   │   ├── helpers.go          # NEW: helper function system
│   │   ├── tps_tracker.go      # NEW: internal TPS tracking
│   │   └── client_test.go      # Modified: new tests
│   ├── config/
│   │   ├── types.go            # Modified: add ProductLimits
│   │   └── parser.go           # Modified: parse limits section
│   └── codegen/
│       ├── generator.go        # Modified: enhanced generator
│       ├── templates.go        # NEW: wrapper templates
│       └── generator_test.go   # NEW: generator tests
├── cmd/
│   └── lcc-codegen/
│       └── main.go             # Modified: use new generator
├── docs/
│   ├── HELPER_FUNCTIONS_GUIDE.md  # NEW
│   ├── MIGRATION_GUIDE_V2.md      # NEW
│   └── REFACTORING_TASK_SDK_ZERO_INTRUSION.md  # This file
└── examples/
    └── zero-intrusion/         # NEW: example project
        ├── main.go
        ├── helpers.go
        └── lcc-features.yaml
```

---

## ✅ Success Checklist

### API Design
- [ ] All limit APIs are product-level (no featureID)
- [ ] Helper function registration works
- [ ] Backward compatibility maintained
- [ ] Clean separation of concerns

### Code Generator
- [ ] Reads YAML with product-level limits
- [ ] Generates wrapper functions
- [ ] Auto-injects license checks
- [ ] Handles helper function calls

### Testing
- [ ] All unit tests pass
- [ ] Integration tests pass
- [ ] Backward compatibility tests pass
- [ ] Performance tests acceptable

### Documentation
- [ ] API documentation updated
- [ ] Migration guide created
- [ ] Helper function guide created
- [ ] Examples provided

---

## 🚀 Getting Started

### Prerequisites
- Go 1.21+
- Access to lcc-sdk repository
- Familiarity with demo app refactoring

### Quick Start
1. Review demo app refactoring:
   ```bash
   cd /home/fila/jqdDev_2025/lcc-demo-app
   cat docs/REFACTORING_COMPLETION_ZERO_INTRUSION.md
   ```

2. Start with Phase 1, Task 1.1:
   ```bash
   cd /home/fila/jqdDev_2025/lcc-sdk
   # Create pkg/client/helpers.go
   ```

3. Follow tasks in order, checking off as you complete

---

## 📞 Questions?

- **Design Questions**: Refer to demo app documentation
- **API Questions**: Check current client.go implementation
- **Architecture**: See ZERO_INTRUSION_COMPARISON.md in demo app

---

**Last Updated**: 2025-01-22  
**Next Review**: After Phase 1 completion
