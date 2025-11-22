# SDK-LCC Server Interaction Diagram

**SDK Version**: 2.0.0 (Zero-Intrusion)  
**Date**: 2025-01-22

---

## Product-Level Check Flow (NEW)

```
┌─────────────┐                              ┌─────────────┐
│             │                              │             │
│  LCC-SDK    │                              │  LCC Server │
│  (Client)   │                              │             │
│             │                              │             │
└──────┬──────┘                              └──────┬──────┘
       │                                            │
       │  1. client.Consume(10)                    │
       │     ↓                                      │
       │  2. checkProductLimits()                  │
       │     ↓                                      │
       │  GET /api/v1/sdk/features/__product__/check
       ├───────────────────────────────────────────→│
       │                                            │
       │                              3. Recognize "__product__"
       │                                 ↓          │
       │                              Check license.productLimits
       │                                 ↓          │
       │                              Return quota, TPS, capacity
       │                                            │
       │  4. Response (product-level limits)       │
       │←───────────────────────────────────────────┤
       │  {                                         │
       │    "feature_id": "__product__",           │
       │    "enabled": true,                        │
       │    "quota_info": {                         │
       │      "limit": 1000,                        │
       │      "used": 150,                          │
       │      "remaining": 850                      │
       │    },                                      │
       │    "max_tps": 100.0,                       │
       │    "max_capacity": 500,                    │
       │    "max_concurrency": 10                   │
       │  }                                         │
       │                                            │
       │  5. reportProductUsage(10)                │
       │     ↓                                      │
       │  POST /api/v1/sdk/usage                   │
       │  {                                         │
       │    "feature_id": "__product__",           │
       │    "count": 10                             │
       │  }                                         │
       ├───────────────────────────────────────────→│
       │                                            │
       │                              6. Increment product quota
       │                                 ↓          │
       │                              quota_used += 10
       │                                            │
       │  7. Success                                │
       │←───────────────────────────────────────────┤
       │                                            │
```

---

## Feature-Level Check Flow (OLD - Still Supported)

```
┌─────────────┐                              ┌─────────────┐
│             │                              │             │
│  LCC-SDK    │                              │  LCC Server │
│  (Client)   │                              │             │
│             │                              │             │
└──────┬──────┘                              └──────┬──────┘
       │                                            │
       │  1. client.CheckFeature("feature-export") │
       │                                            │
       │  GET /api/v1/sdk/features/feature-export/check
       ├───────────────────────────────────────────→│
       │                                            │
       │                              2. Find feature in license
       │                                 ↓          │
       │                              Check license.features["feature-export"]
       │                                 ↓          │
       │                              Return feature-specific limits
       │                                            │
       │  3. Response (feature-level)              │
       │←───────────────────────────────────────────┤
       │  {                                         │
       │    "feature_id": "feature-export",        │
       │    "enabled": true,                        │
       │    "quota_info": { ... }                   │
       │  }                                         │
       │                                            │
```

---

## Complete Workflow: Zero-Intrusion API

### Step 1: Application Startup

```
┌──────────────┐
│ Application  │
└──────┬───────┘
       │
       │ 1. Create LCC client
       ↓
   cfg := &config.SDKConfig{...}
   client := NewClient(cfg)
       │
       │ 2. Register helpers
       ↓
   helpers := &HelperFunctions{
     QuotaConsumer: calculateBatchSize,
     CapacityCounter: countActiveUsers,
   }
   client.RegisterHelpers(helpers)
       │
       │ 3. Register with LCC
       ↓
   client.Register()
       │
       ↓
   ┌─────────────┐
   │ LCC Server  │
   └─────────────┘
```

### Step 2: Runtime Check (Product-Level)

```
┌──────────────┐        ┌─────────────┐        ┌─────────────┐
│ Business     │        │  LCC-SDK    │        │ LCC Server  │
│ Logic        │        │  (Client)   │        │             │
└──────┬───────┘        └──────┬──────┘        └──────┬──────┘
       │                       │                       │
       │ ProcessBatch(100)     │                       │
       ├──────────────────────→│                       │
       │                       │                       │
       │                       │ 1. Consume(100)      │
       │                       │    ↓                  │
       │                       │ checkProductLimits() │
       │                       ├──────────────────────→│
       │                       │ GET __product__/check│
       │                       │                       │
       │                       │←──────────────────────┤
       │                       │ {quota: ok}          │
       │                       │                       │
       │                       │ 2. reportUsage(100)  │
       │                       ├──────────────────────→│
       │                       │ POST usage           │
       │                       │                       │
       │                       │←──────────────────────┤
       │                       │ Success              │
       │                       │                       │
       │←──────────────────────┤                       │
       │ OK                    │                       │
       │                       │                       │
```

---

## Data Flow: License Structure

### Product-Level License (NEW)

```
┌────────────────────────────────────────────┐
│              License File                  │
├────────────────────────────────────────────┤
│                                            │
│  licenseId: "lic-123"                     │
│  productId: "my-app"                      │
│  valid: true                               │
│                                            │
│  planInfo:                                 │
│    ┌──────────────────────────────────┐   │
│    │  productLimits:                  │   │
│    │    quota:                        │   │
│    │      max: 1000    ◄──────────────┼───┼─── SDK reads this
│    │      used: 150                   │   │    for product-level
│    │      remaining: 850              │   │    checks
│    │    maxTPS: 100.0                 │   │
│    │    maxCapacity: 500              │   │
│    │    maxConcurrency: 10            │   │
│    └──────────────────────────────────┘   │
│                                            │
│    features:                               │
│      feature-export: {enabled: true}      │
│      feature-analytics: {enabled: true}   │
│                                            │
└────────────────────────────────────────────┘
```

### Feature-Level License (OLD)

```
┌────────────────────────────────────────────┐
│              License File                  │
├────────────────────────────────────────────┤
│                                            │
│  licenseId: "lic-123"                     │
│  productId: "my-app"                      │
│                                            │
│  planInfo:                                 │
│    features:                               │
│      feature-export:                       │
│        enabled: true                       │
│        quota: {daily: 500} ◄───────────────┼─── SDK reads this
│      feature-analytics:                    │    for feature-level
│        enabled: true                       │    checks
│        quota: {daily: 300}                 │
│                                            │
└────────────────────────────────────────────┘
```

---

## Comparison: API Request Patterns

### Product-Level (Zero-Intrusion)

```
Business Code (Clean):
┌─────────────────────────┐
│ func ExportData() {     │
│   // No license code!   │
│   return doExport()     │
│ }                       │
└─────────────────────────┘
         ↓
    Code Generator Wraps:
┌─────────────────────────┐
│ func ExportData() {     │
│   // Auto-injected:     │
│   allowed, _, err :=    │
│     client.Consume(1)   │  ◄── Product-level
│   if !allowed {         │      No featureID
│     return error        │
│   }                     │
│   return doExport()     │
│ }                       │
└─────────────────────────┘
         ↓
    SDK Request:
┌─────────────────────────┐
│ GET /features/          │
│   __product__/check     │  ◄── Special ID
└─────────────────────────┘
```

### Feature-Level (Old)

```
Business Code (Invasive):
┌─────────────────────────┐
│ func ExportData() {     │
│   // License check:     │
│   status, err :=        │
│     client.CheckFeature(│
│       "feature-export"  │  ◄── featureID required
│     )                   │
│   if !status.Enabled {  │
│     return error        │
│   }                     │
│   return doExport()     │
│ }                       │
└─────────────────────────┘
         ↓
    SDK Request:
┌─────────────────────────┐
│ GET /features/          │
│   feature-export/check  │  ◄── Feature-specific
└─────────────────────────┘
```

---

## Server-Side Decision Tree

```
                     Request received
                           ↓
                  ┌────────┴────────┐
                  │  feature_id?    │
                  └────────┬────────┘
                           ↓
              ┌────────────┴────────────┐
              │                         │
    "__product__"              Other feature ID
              │                         │
              ↓                         ↓
    ┌─────────────────────┐   ┌─────────────────────┐
    │ Product-Level Check │   │ Feature-Level Check │
    ├─────────────────────┤   ├─────────────────────┤
    │ 1. Get license      │   │ 1. Get license      │
    │ 2. Read productLimits│  │ 2. Find feature     │
    │ 3. Return:          │   │ 3. Return:          │
    │   - quota (shared)  │   │   - feature status  │
    │   - max_tps         │   │   - feature quota   │
    │   - max_capacity    │   │   - feature limits  │
    │   - max_concurrency │   │                     │
    └─────────────────────┘   └─────────────────────┘
              │                         │
              └────────────┬────────────┘
                           ↓
                     Return response
```

---

## Migration Path

### Phase 1: Both APIs Supported

```
┌──────────────────────────────────────────────┐
│              LCC Server                      │
├──────────────────────────────────────────────┤
│                                              │
│  /features/__product__/check   ◄─── NEW     │
│      └→ Returns productLimits               │
│                                              │
│  /features/{featureID}/check  ◄─── OLD      │
│      └→ Returns feature limits              │
│                                              │
│  Both work simultaneously!                   │
│                                              │
└──────────────────────────────────────────────┘
          ↓                    ↓
    New SDK v2.0          Old SDK v1.x
    (Zero-intrusion)      (Feature-level)
```

### Phase 2: Gradual Migration

```
    Time →
    
    Old SDK v1.x:  ████████████░░░░░░░░
                   (gradually phased out)
    
    New SDK v2.0:  ░░░░░░░░████████████
                   (gradually adopted)
    
    Server:        Both APIs supported throughout
```

---

## Summary

### Key Points for LCC Server

1. **Recognize `__product__`** as special feature ID
2. **Return product-level limits** when `__product__` is requested
3. **Track quota at product level** for `__product__` usage reports
4. **Maintain backward compatibility** with feature-level checks
5. **Update license format** to include `productLimits` section

### Benefits

✅ **Zero-intrusion**: Business code stays clean  
✅ **Product-level control**: Unified limits  
✅ **Backward compatible**: Old API still works  
✅ **Flexible**: Supports both patterns during migration  

---

**Last Updated**: 2025-01-22
