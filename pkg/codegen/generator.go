package codegen

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"text/template"

	"github.com/yourorg/lcc-sdk/pkg/config"
)

// Generator generates wrapper code for license-protected functions
type Generator struct {
	manifest *config.Manifest
}

// NewGenerator creates a new code generator
func NewGenerator(manifest *config.Manifest) *Generator {
	return &Generator{
		manifest: manifest,
	}
}

// Generate generates wrapper code for all features in the manifest
func (g *Generator) Generate(outputDir string) error {
	// Group features by package
	packageFeatures := make(map[string][]config.FeatureConfig)
	for _, feature := range g.manifest.Features {
		pkg := feature.Intercept.Package
		packageFeatures[pkg] = append(packageFeatures[pkg], feature)
	}

	// Generate code for each package
	for pkgPath, features := range packageFeatures {
		if err := g.generatePackage(pkgPath, features, outputDir); err != nil {
			return fmt.Errorf("failed to generate package %s: %w", pkgPath, err)
		}
	}

	return nil
}

// generatePackage generates wrapper code for a specific package
func (g *Generator) generatePackage(pkgPath string, features []config.FeatureConfig, outputDir string) error {
	// Extract package name from path
	pkgName := filepath.Base(pkgPath)

	// Build function templates
	var functions []FunctionTemplate
	for _, feature := range features {
		funcTemplate, err := g.buildFunctionTemplate(feature)
		if err != nil {
			return fmt.Errorf("failed to build template for feature %s: %w", feature.ID, err)
		}
		functions = append(functions, funcTemplate)
	}

	// Create package template
	pkgTemplate := PackageTemplate{
		Package:   pkgName,
		Functions: functions,
	}

	// Generate code
	code, err := g.renderTemplate(pkgTemplate)
	if err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	// Format code
	formatted, err := format.Source([]byte(code))
	if err != nil {
		// If formatting fails, save unformatted for debugging
		fmt.Printf("Warning: failed to format code: %v\n", err)
		formatted = []byte(code)
	}

	// Write to file
	outputPath := filepath.Join(outputDir, pkgName, "lcc_gen.go")
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	if err := os.WriteFile(outputPath, formatted, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	fmt.Printf("Generated: %s\n", outputPath)
	return nil
}

// buildFunctionTemplate builds a function template from feature config
func (g *Generator) buildFunctionTemplate(feature config.FeatureConfig) (FunctionTemplate, error) {
	funcName := feature.Intercept.Function

	// Build signature (simplified - assumes no complex types)
	signature := fmt.Sprintf("func %s(args ...interface{}) (interface{}, error)", funcName)

	// Build original call
	originalCall := fmt.Sprintf("return %s_Original(args...)", funcName)

	// Build fallback call if exists
	var fallbackCall string
	hasFallback := feature.Fallback != nil
	if hasFallback {
		fallbackFunc := feature.Fallback.Function
		fallbackCall = fmt.Sprintf("return %s(args...)", fallbackFunc)
	}

	// Build error return
	errorReturn := `return nil, fmt.Errorf("feature not licensed")`

	return FunctionTemplate{
		OriginalName: funcName,
		Signature:    signature,
		FeatureID:    feature.ID,
		HasFallback:  hasFallback,
		FallbackCall: fallbackCall,
		ErrorReturn:  errorReturn,
		OriginalCall: originalCall,
	}, nil
}

// renderTemplate renders the code template
func (g *Generator) renderTemplate(pkgTemplate PackageTemplate) (string, error) {
	tmpl, err := template.New("wrapper").Parse(WrapperTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, pkgTemplate); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// GenerateForFeature generates wrapper code for a single feature (utility function)
func GenerateForFeature(feature *config.FeatureConfig, outputPath string) error {
	manifest := &config.Manifest{
		Features: []config.FeatureConfig{*feature},
	}

	gen := NewGenerator(manifest)
	return gen.Generate(filepath.Dir(outputPath))
}

// GenerateZeroIntrusion generates zero-intrusion wrapper code using product-level API
// This method uses ProductLimits from the manifest instead of feature-level limits
func (g *Generator) GenerateZeroIntrusion(outputDir string) error {
	if g.manifest.SDK.Limits == nil {
		return fmt.Errorf("no product limits defined in manifest (required for zero-intrusion mode)")
	}

	// Group features by package
	packageFeatures := make(map[string][]config.FeatureConfig)
	for _, feature := range g.manifest.Features {
		pkg := feature.Intercept.Package
		packageFeatures[pkg] = append(packageFeatures[pkg], feature)
	}

	// Generate code for each package
	for pkgPath, features := range packageFeatures {
		if err := g.generateZeroIntrusionPackage(pkgPath, features, outputDir); err != nil {
			return fmt.Errorf("failed to generate zero-intrusion package %s: %w", pkgPath, err)
		}
	}

	return nil
}

// generateZeroIntrusionPackage generates zero-intrusion wrapper code for a package
func (g *Generator) generateZeroIntrusionPackage(pkgPath string, features []config.FeatureConfig, outputDir string) error {
	pkgName := filepath.Base(pkgPath)

	// Build function templates using product-level limits
	var functions []ZeroIntrusionFunctionTemplate
	for _, feature := range features {
		funcTemplate, err := g.buildZeroIntrusionTemplate(feature)
		if err != nil {
			return fmt.Errorf("failed to build zero-intrusion template for feature %s: %w", feature.ID, err)
		}
		functions = append(functions, funcTemplate)
	}

	// Create package template
	pkgTemplate := ZeroIntrusionPackageTemplate{
		Package:   pkgName,
		Functions: functions,
	}

	// Generate code using zero-intrusion template
	code, err := g.renderZeroIntrusionTemplate(pkgTemplate)
	if err != nil {
		return fmt.Errorf("failed to render zero-intrusion template: %w", err)
	}

	// Format code
	formatted, err := format.Source([]byte(code))
	if err != nil {
		fmt.Printf("Warning: failed to format code: %v\n", err)
		formatted = []byte(code)
	}

	// Write to file
	outputPath := filepath.Join(outputDir, pkgName, "lcc_gen_zero_intrusion.go")
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	if err := os.WriteFile(outputPath, formatted, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	fmt.Printf("Generated (zero-intrusion): %s\n", outputPath)
	return nil
}

// buildZeroIntrusionTemplate builds a zero-intrusion function template
func (g *Generator) buildZeroIntrusionTemplate(feature config.FeatureConfig) (ZeroIntrusionFunctionTemplate, error) {
	funcName := feature.Intercept.Function
	limits := g.manifest.SDK.Limits

	// Build signature (simplified)
	signature := fmt.Sprintf("func %s(args ...interface{}) (interface{}, error)", funcName)

	// Build original call
	originalCall := fmt.Sprintf("return %s_Original(args...)", funcName)

	// Build fallback call if exists
	var fallbackCall string
	hasFallback := feature.Fallback != nil
	if hasFallback {
		fallbackFunc := feature.Fallback.Function
		fallbackCall = fmt.Sprintf("return %s(args...)", fallbackFunc)
	}

	// Build error return
	errorReturn := `return nil, fmt.Errorf("license limit exceeded")`

	// Determine which limits to inject based on ProductLimits
	hasConcurrency := limits.MaxConcurrency > 0
	hasQuota := limits.Quota != nil && limits.Quota.Max > 0
	hasTPS := limits.MaxTPS > 0
	hasCapacity := limits.MaxCapacity > 0

	return ZeroIntrusionFunctionTemplate{
		OriginalName:   funcName,
		Signature:      signature,
		HasConcurrency: hasConcurrency,
		HasQuota:       hasQuota,
		HasTPS:         hasTPS,
		HasCapacity:    hasCapacity,
		QuotaConsumer:  limits.Consumer,
		PassArgs:       limits.Consumer != "", // Pass args if custom consumer specified
		HasFallback:    hasFallback,
		FallbackCall:   fallbackCall,
		ErrorReturn:    errorReturn,
		OriginalCall:   originalCall,
	}, nil
}

// renderZeroIntrusionTemplate renders the zero-intrusion code template
func (g *Generator) renderZeroIntrusionTemplate(pkgTemplate ZeroIntrusionPackageTemplate) (string, error) {
	tmpl, err := template.New("zero_intrusion_wrapper").Parse(ZeroIntrusionWrapperTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse zero-intrusion template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, pkgTemplate); err != nil {
		return "", fmt.Errorf("failed to execute zero-intrusion template: %w", err)
	}

	return buf.String(), nil
}
