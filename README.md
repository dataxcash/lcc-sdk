# LCC SDK

Official Go SDK for integrating applications with LCC (License Control Center).

## Features

- 🔐 **Zero-configuration authentication** - Self-signed RSA key pairs, no pre-registration needed
- 📝 **Declarative license control** - YAML-based feature protection
- 🔧 **Code generation** - Automatic function wrapping with minimal code intrusion
- 📊 **Automatic usage reporting** - Built-in quota management and usage tracking
- 🎯 **Graceful degradation** - Automatic fallback to basic features
- 🚀 **Production-ready** - Caching, retry logic, and offline support

## Quick Start

### Installation

```bash
go get github.com/yourorg/lcc-sdk
```

### Basic Usage

1. **Create feature manifest:**

```yaml
# lcc-features.yaml
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
```

2. **Add to build process:**

```makefile
# Makefile
.PHONY: generate
generate:
	go generate ./...

.PHONY: build
build: generate
	go build -o myapp ./cmd/myapp
```

3. **Initialize SDK in your app:**

```go
package main

import (
    _ "github.com/yourorg/lcc-sdk/auto"
)

func main() {
    // Your code - SDK works transparently
    result := analytics.AdvancedAnalytics(data)
}
```

## Documentation

- [Getting Started Guide](docs/getting-started.md)
- [Configuration Reference](docs/configuration.md)
- [API Documentation](docs/api-reference.md)
- [Code Generation Guide](docs/codegen.md)

## Examples

See [lcc-demo-app](https://github.com/yourorg/lcc-demo-app) for a complete working example.

## Architecture

```
Application Code
       ↓
Feature Manifest (YAML)
       ↓
Code Generator (lcc-codegen)
       ↓
Generated Wrappers
       ↓
LCC Client (runtime)
       ↓
LCC Server (license verification)
```

## Development

```bash
# Clone repository
git clone https://github.com/yourorg/lcc-sdk
cd lcc-sdk

# Install dependencies
go mod download

# Run tests
make test

# Build CLI tools
make build
```

## Related Projects

- [lcc-demo-app](https://github.com/yourorg/lcc-demo-app) - Complete demo application
- [LCC Server](https://yourcompany.com/lcc) - Commercial license server

## License

MIT License - see [LICENSE](LICENSE) for details.

## Support

- Documentation: https://docs.yourcompany.com/lcc-sdk
- Issues: https://github.com/yourorg/lcc-sdk/issues
- Community: https://community.yourcompany.com
