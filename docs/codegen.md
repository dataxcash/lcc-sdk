# Code Generation Guide

This guide explains how to use the LCC SDK's code generator to protect your
functions with license checks.

## 1. Overview

The `lcc-codegen` tool reads a manifest (`lcc-features.yaml`) and generates
wrapper functions that:

- Check license status for a feature
- Apply quota, tier, and on-deny behavior
- Optionally route to a fallback implementation

Generated files follow the pattern `lcc_gen.go` inside the target packages.

## 2. CLI Usage

Basic usage:

```bash
lcc-codegen --config lcc-features.yaml --output ./
```

Flags:

- `--config` (string, required): Path to the manifest YAML file.
- `--output` (string, required): Root directory where generated code should be placed.

## 3. Manifest-Driven Generation

The generator groups features by `intercept.package` and emits one
`lcc_gen.go` per package.

Example manifest snippet:

```yaml
features:
  - id: advanced_analytics
    name: "Advanced Analytics"
    intercept:
      package: "myapp/analytics"
      function: "AdvancedAnalytics"
    fallback:
      package: "myapp/analytics"
      function: "BasicAnalytics"
    on_deny:
      action: fallback
```

From this, `lcc-codegen` will:

1. Create/ensure `myapp/analytics` exists under the output directory.
2. Generate a wrapper for `AdvancedAnalytics` in `lcc_gen.go`.
3. Add logic to check the feature status and call either the original or
   fallback function.

## 4. Generated Code Shape

Internally, the generator uses a template similar to:

- Wrapper signature using `func <Name>(args ...interface{}) (interface{}, error)`
- Calls to `client.CheckFeature` or higher-level helpers before delegating
  to the original function.

See `pkg/codegen/generator.go` and `pkg/codegen/templates.go` for the exact
implementation.

## 5. Regeneration and Cleanup

- Regenerate whenever `lcc-features.yaml` changes.
- Generated files are named `lcc_gen.go` and are safe to delete and regenerate.
- The `.gitignore` in this repository already excludes `**/lcc_gen.go` by
  default; applications can choose whether to commit generated code or not.

## 6. Integration Pattern

A common build pattern is:

```makefile
.PHONY: generate
generate:
	lcc-codegen --config lcc-features.yaml --output ./

.PHONY: build
build: generate
	go build ./cmd/myapp
```

This ensures that wrappers are always up to date before building your
application.
