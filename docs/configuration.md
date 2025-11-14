# Configuration Reference

This document describes the configuration model used by the LCC SDK,
including the YAML manifest (`lcc-features.yaml`) and the `SDKConfig`
structure.

## 1. Manifest Structure

Top-level structure:

```yaml
sdk:
  # Global SDK configuration

features:
  - id: ...      # FeatureConfig
    name: ...
    ...
```

### 1.1 `sdk` (SDKConfig)

Mapped to `config.SDKConfig`:

```yaml
sdk:
  lcc_url: "http://localhost:7086"   # Required
  product_id: "my-app"              # Required
  product_version: "1.0.0"          # Required
  check_interval: 30s                # Optional (Go duration)
  cache_ttl: 10s                     # Optional (Go duration)
  fail_open: false                   # Optional, default false
  timeout: 5s                        # Optional (Go duration)
  max_retries: 3                     # Optional, default 3
```

See `pkg/config/types.go` for defaults.

### 1.2 `features[]` (FeatureConfig)

Each entry describes one protected feature:

```yaml
features:
  - id: advanced_analytics          # Required, unique
    name: "Advanced Analytics"      # Required
    description: "ML-powered analytics"  # Optional
    tier: professional              # Optional (free/basic/pro/ent)
    required: false                 # Optional, default false

    intercept:
      package: "myapp/analytics"    # Required
      function: "AdvancedAnalytics" # OR pattern: "^Advanced.*" (one required)

    fallback:
      package: "myapp/analytics"    # Optional
      function: "BasicAnalytics"

    quota:
      limit: 1000                   # > 0
      period: daily                 # daily/hourly/monthly/minute
      reset_time: "00:00"          # Optional (HH:MM)

    condition:
      type: runtime                 # runtime/static
      check: "user.plan == 'pro'"   # Expression evaluated by app/integration

    on_deny:
      action: fallback              # fallback/error/warn/filter
      message: "Feature not licensed"  # Optional
      code: "ERR_FEATURE_DENIED"       # Optional
```

Validation rules are implemented in `config.Manifest.Validate()` and
`FeatureConfig.Validate()`.

## 2. SDKConfig Fields

From `pkg/config/types.go`:

- `LCCURL` (string, required)
- `ProductID` (string, required)
- `ProductVersion` (string, required)
- `CheckInterval` (time.Duration, default 30s)
- `CacheTTL` (time.Duration, default 10s)
- `FailOpen` (bool, default false)
- `Timeout` (time.Duration, default 5s)
- `MaxRetries` (int, default 3)

These values control how often the SDK refreshes feature state, how long
results are cached, behavior on failure (fail-open vs fail-closed), and HTTP
client behavior.

## 3. Defaults Helper

`config.GetDefaults()` returns a manifest populated with reasonable defaults
for the `sdk` section:

```go
m := config.GetDefaults()
// m.SDK is pre-filled with LCCURL=http://localhost:7086 and other defaults
```

You can use this when building manifests programmatically.

## 4. Validation Errors

Configuration validation errors are returned as `*config.ValidationError`:

```go
err := manifest.Validate()
if err != nil {
    if vErr, ok := err.(*config.ValidationError); ok {
        // vErr.Field and vErr.Message describe the issue
    }
}
```

The error message is formatted as:

```text
validation error in field '<field>': <message>
```
