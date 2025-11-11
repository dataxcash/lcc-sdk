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
