package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadManifest loads and parses the lcc-features.yaml file
func LoadManifest(path string) (*Manifest, error) {
	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest file: %w", err)
	}

	// Parse YAML
	manifest := GetDefaults()
	if err := yaml.Unmarshal(data, manifest); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Validate
	if err := manifest.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	return manifest, nil
}

// LoadManifestFromBytes loads manifest from byte slice
func LoadManifestFromBytes(data []byte) (*Manifest, error) {
	manifest := GetDefaults()
	if err := yaml.Unmarshal(data, manifest); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	if err := manifest.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	return manifest, nil
}

// SaveManifest saves manifest to file
func SaveManifest(manifest *Manifest, path string) error {
	// Validate before saving
	if err := manifest.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	// Write to file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// ValidateManifest validates a manifest without loading from file
func ValidateManifest(manifest *Manifest) error {
	return manifest.Validate()
}

// FindFeature finds a feature by ID
func (m *Manifest) FindFeature(id string) *FeatureConfig {
	for i := range m.Features {
		if m.Features[i].ID == id {
			return &m.Features[i]
		}
	}
	return nil
}

// HasFeature checks if a feature exists
func (m *Manifest) HasFeature(id string) bool {
	return m.FindFeature(id) != nil
}

// GetFeatureIDs returns all feature IDs
func (m *Manifest) GetFeatureIDs() []string {
	ids := make([]string, len(m.Features))
	for i, feature := range m.Features {
		ids[i] = feature.ID
	}
	return ids
}

// GetFeaturesByTier returns features of a specific tier
func (m *Manifest) GetFeaturesByTier(tier string) []FeatureConfig {
	var features []FeatureConfig
	for _, feature := range m.Features {
		if feature.Tier == tier {
			features = append(features, feature)
		}
	}
	return features
}

// GetRequiredFeatures returns all required features
func (m *Manifest) GetRequiredFeatures() []FeatureConfig {
	var features []FeatureConfig
	for _, feature := range m.Features {
		if feature.Required {
			features = append(features, feature)
		}
	}
	return features
}
