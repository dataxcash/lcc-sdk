# Getting Started with LCC SDK

This guide shows how to install, configure, and use the LCC SDK in a Go
application.

## 1. Install

Add the SDK module to your project:

```bash
go get github.com/yourorg/lcc-sdk
```

> Note: The module path matches the `module` directive in `go.mod`.

## 2. Define a Feature Manifest

Create an `lcc-features.yaml` file describing your product and protected
features:

```yaml
sdk:
  lcc_url: "http://localhost:7086"
  product_id: "my-app"
  product_version: "1.0.0"

features:
  - id: advanced_analytics
    name: "Advanced Analytics"
    tier: professional
    intercept:
      package: "myapp/analytics"
      function: "AdvancedAnalytics"
    fallback:
      package: "myapp/analytics"
      function: "BasicAnalytics"
    quota:
      limit: 1000
      period: daily
```

## 3. Run Code Generation

Use `lcc-codegen` (CLI) to generate wrappers based on the manifest:

```bash
lcc-codegen --config lcc-features.yaml --output ./
```

This will create `lcc_gen.go` files in the appropriate packages.

## 4. Initialize SDK in Your App

In your main package, import the generated wrappers and initialize the SDK
client:

```go
package main

import (
    "log"
    "time"

    "github.com/yourorg/lcc-sdk/pkg/client"
    "github.com/yourorg/lcc-sdk/pkg/config"
)

func main() {
    cfg := &config.SDKConfig{
        LCCURL:         "https://localhost:8088",
        ProductID:      "my-app",
        ProductVersion: "1.0.0",
        Timeout:        5 * time.Second,
        CacheTTL:       10 * time.Second,
    }

    c, err := client.NewClient(cfg)
    if err != nil {
        log.Fatalf("failed to create LCC client: %v", err)
    }
    defer c.Close()

    if err := c.Register(); err != nil {
        log.Fatalf("failed to register with LCC: %v", err)
    }

    // Call your protected features as usual
}
```

## 5. Next Steps

- See [Configuration Reference](configuration.md) for all manifest options.
- See [API Documentation](api-reference.md) for client APIs.
- See [Code Generation Guide](codegen.md) for advanced codegen usage.
