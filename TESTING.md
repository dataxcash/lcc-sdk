# Testing LCC SDK

## Quick Start

### Run All Tests

```bash
make test
```

### Build Demo

```bash
make demo
```

### Test Zero-Intrusion API (Standalone)

Test the zero-intrusion API **without requiring LCC server**:

```bash
make test-demo
```

This will run a standalone test that validates:
- ✅ Helper function creation and validation
- ✅ TPS tracker functionality
- ✅ API type definitions
- ✅ Zero-intrusion API structure

### Run Demo with LCC Server

If you have LCC server running on `localhost:7086`:

```bash
make run-demo
```

Or manually:

```bash
cd examples/zero-intrusion
./demo
```

---

## Available Make Targets

Run `make help` to see all available commands:

```bash
make help
```

Output:
```
LCC SDK Makefile Commands:

  make build       - Build lcc-codegen and lcc-sdk binaries
  make test        - Run all unit tests
  make demo        - Build zero-intrusion demo
  make run-demo    - Build and run demo (requires LCC server)
  make test-demo   - Run standalone demo test (no LCC server)
  make clean       - Clean build artifacts
  make install     - Install binaries
  make fmt         - Format code
  make lint        - Run linter
  make deps        - Download and tidy dependencies
```

---

## Testing Without LCC Server

The SDK provides standalone tests that don't require a running LCC server:

### 1. Standalone API Test

```bash
cd examples/zero-intrusion
go run test_standalone.go
```

Expected output:
```
=== LCC SDK Zero-Intrusion API Test (Standalone) ===

✅ Test 1: Creating helper functions

✅ Test 2: Validating helper functions
   - All helpers valid

✅ Test 3: Testing TPS tracker
   - TPS tracker working: 3.00 TPS

✅ Test 4: Testing API types
   - Release function called

=== All Standalone Tests Passed! ===
```

### 2. Unit Tests

```bash
make test
```

This runs all package-level unit tests.

---

## Testing With LCC Server

### Prerequisites

1. LCC server running on `localhost:7086`
2. Valid license configured

### Run Integration Tests

```bash
cd examples/zero-intrusion
./demo
```

This will:
1. Create LCC client
2. Register helper functions
3. Register with LCC server
4. Run example API calls:
   - Simple quota consumption
   - Quota with helper function
   - TPS check
   - Capacity check
   - Concurrency control

---

## Expected Behavior

### With LCC Server Running

```
✅ Example 1: Consumed 1 unit, 999 remaining
✅ Example 2: Consumed 10 units, 989 remaining
✅ Example 3: TPS check passed (max=100.00)
✅ Example 4: Capacity check passed (max=500)
✅ Example 5: Slot acquired, performing operation...
All examples completed successfully!
```

### Without LCC Server

```
Failed to register with LCC: request failed: Post "http://localhost:7086/api/v1/sdk/register": dial tcp [::1]:7086: connect: connection refused
```

This is expected! Use `make test-demo` instead for standalone testing.

---

## Test Coverage

Generate test coverage report:

```bash
make test-coverage
```

This creates:
- `coverage.out` - Coverage data
- `coverage.html` - HTML coverage report

View coverage:
```bash
open coverage.html  # macOS
xdg-open coverage.html  # Linux
```

---

## Troubleshooting

### Error: "make: *** No rule to make target 'demo'"

**Solution**: Update your repository:
```bash
git pull
```

### Error: "connection refused" when running demo

**Cause**: LCC server is not running.

**Solutions**:
1. Start LCC server on `localhost:7086`
2. Or use standalone test: `make test-demo`

### Error: "helper validation failed"

**Cause**: CapacityCounter helper is required.

**Solution**: Always provide CapacityCounter:
```go
helpers := &client.HelperFunctions{
    CapacityCounter: func() int { return 0 },  // Stub if not using capacity
}
```

---

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Test
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - uses: actions/setup-go@v2
        with:
          go-version: '1.21'
      - run: make deps
      - run: make test
      - run: make test-demo
      - run: make build
```

---

## See Also

- [Migration Guide](docs/MIGRATION_GUIDE_ZERO_INTRUSION.md)
- [Helper Functions Guide](docs/HELPER_FUNCTIONS_GUIDE.md)
- [Example Code](examples/zero-intrusion/)
