# API Documentation

This document summarizes the primary Go APIs exposed by the LCC SDK.

For full details, consult the GoDoc-style comments in the source under `pkg/`.

## Package `config`

### Types

- `type Manifest struct`
  - Represents the full `lcc-features.yaml` manifest.
  - Fields:
    - `SDK SDKConfig`
    - `Features []FeatureConfig`

- `type SDKConfig struct`
  - Global SDK configuration, usually loaded from the `sdk` section of the manifest.

- `type FeatureConfig struct`
  - Configuration for a single protected feature.

- `type InterceptConfig struct`
- `type QuotaConfig struct`
- `type ConditionConfig struct`
- `type OnDenyConfig struct`
- `type ValidationError struct`

### Key Functions/Methods

- `func (m *Manifest) Validate() error`
- `func (c *SDKConfig) Validate() error`
- `func (f *FeatureConfig) Validate() error`
- `func (i *InterceptConfig) Validate() error`
- `func (q *QuotaConfig) Validate() error`
- `func (o *OnDenyConfig) Validate() error`
- `func GetDefaults() *Manifest`

These functions enforce required fields, valid values, and provide helpful
validation errors.

## Package `client`

### Types

- `type Client struct`
  - Main entry point for interacting with the LCC server.

- `type FeatureStatus struct`
  - Represents the result of a feature check.
  - Fields:
    - `Enabled bool`
    - `Reason string`
    - `Quota *QuotaInfo`
    - `MaxCapacity int`
    - `MaxTPS float64`
    - `MaxConcurrency int`

- `type QuotaInfo struct`
  - Mirrors server-side quota information.

### Key Functions/Methods

- `func NewClient(cfg *config.SDKConfig) (*Client, error)`
- `func (c *Client) Register() error`
- `func (c *Client) CheckFeature(featureID string) (*FeatureStatus, error)`
- `func (c *Client) Consume(featureID string, amount int, meta map[string]any) (bool, int, string, error)`
- `func (c *Client) CheckCapacity(featureID string, currentUsed int) (bool, int, string, error)`
- `func (c *Client) CheckTPS(featureID string, currentTPS float64) (bool, float64, string, error)`
- `func (c *Client) AcquireSlot(featureID string, meta map[string]any) (release func(), allowed bool, reason string, err error)`
- `func (c *Client) ReportUsage(featureID string, amount float64) error`
- `func (c *Client) GetInstanceID() string`

> Note: Some methods, such as `CheckTPS` and `AcquireSlot`, are designed to
> work with application-provided metrics (current TPS, concurrent jobs/users).

## Package `codegen`

### Types

- `type Generator struct`
  - Responsible for generating wrapper code for license-protected functions.

### Key Functions/Methods

- `func NewGenerator(manifest *config.Manifest) *Generator`
- `func (g *Generator) Generate(outputDir string) error`
- `func GenerateForFeature(feature *config.FeatureConfig, outputPath string) error`

The generator groups features by package and emits `lcc_gen.go` files that wrap
original functions with license checks and optional fallbacks.

## Package `auth`

The `auth` package contains internal helpers for key management and request
signing (RSA key pair generation, request signatures). These are generally not
used directly by applications; they are used internally by `client.Client`.

## Examples

For end-to-end usage examples, see:

- `examples/` in this repository
- The standalone demo application: `lcc-demo-app`
