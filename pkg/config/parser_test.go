package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadManifest(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr bool
	}{
		{
			name: "valid manifest",
			yaml: `
sdk:
  lcc_url: "http://localhost:7086"
  product_id: "test-app"
  product_version: "1.0.0"

features:
  - id: test_feature
    name: "Test Feature"
    tier: professional
    intercept:
      package: "github.com/test/app"
      function: "TestFunc"
`,
			wantErr: false,
		},
		{
			name: "missing product_id",
			yaml: `
sdk:
  lcc_url: "http://localhost:7086"
  product_version: "1.0.0"

features: []
`,
			wantErr: true,
		},
		{
			name: "duplicate feature IDs",
			yaml: `
sdk:
  lcc_url: "http://localhost:7086"
  product_id: "test-app"
  product_version: "1.0.0"

features:
  - id: feature1
    name: "Feature 1"
    tier: basic
    intercept:
      package: "test"
      function: "Func1"
  - id: feature1
    name: "Feature 1 Duplicate"
    tier: basic
    intercept:
      package: "test"
      function: "Func2"
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file
			tmpfile, err := os.CreateTemp("", "test-*.yaml")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(tmpfile.Name())

			// Write YAML
			if _, err := tmpfile.Write([]byte(tt.yaml)); err != nil {
				t.Fatal(err)
			}
			tmpfile.Close()

			// Load manifest
			_, err = LoadManifest(tmpfile.Name())
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadManifest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadManifestFromBytes(t *testing.T) {
	validYAML := []byte(`
sdk:
  lcc_url: "http://localhost:7086"
  product_id: "test-app"
  product_version: "1.0.0"

features:
  - id: test_feature
    name: "Test Feature"
    tier: professional
    intercept:
      package: "test"
      function: "TestFunc"
`)

	manifest, err := LoadManifestFromBytes(validYAML)
	if err != nil {
		t.Fatalf("LoadManifestFromBytes() error = %v", err)
	}

	if manifest.SDK.ProductID != "test-app" {
		t.Errorf("ProductID = %v, want test-app", manifest.SDK.ProductID)
	}

	if len(manifest.Features) != 1 {
		t.Errorf("Features count = %v, want 1", len(manifest.Features))
	}
}

func TestSaveManifest(t *testing.T) {
	manifest := &Manifest{
		SDK: SDKConfig{
			LCCURL:         "http://localhost:7086",
			ProductID:      "test-app",
			ProductVersion: "1.0.0",
			CheckInterval:  30 * time.Second,
			CacheTTL:       10 * time.Second,
			Timeout:        5 * time.Second,
			MaxRetries:     3,
		},
		Features: []FeatureConfig{
			{
				ID:   "test_feature",
				Name: "Test Feature",
				Tier: "professional",
				Intercept: InterceptConfig{
					Package:  "test",
					Function: "TestFunc",
				},
			},
		},
	}

	tmpfile := filepath.Join(os.TempDir(), "test-save.yaml")
	defer os.Remove(tmpfile)

	if err := SaveManifest(manifest, tmpfile); err != nil {
		t.Fatalf("SaveManifest() error = %v", err)
	}

	// Load it back
	loaded, err := LoadManifest(tmpfile)
	if err != nil {
		t.Fatalf("LoadManifest() error = %v", err)
	}

	if loaded.SDK.ProductID != manifest.SDK.ProductID {
		t.Errorf("ProductID = %v, want %v", loaded.SDK.ProductID, manifest.SDK.ProductID)
	}
}

func TestManifest_FindFeature(t *testing.T) {
	manifest := &Manifest{
		SDK: SDKConfig{
			LCCURL:         "http://localhost:7086",
			ProductID:      "test",
			ProductVersion: "1.0.0",
		},
		Features: []FeatureConfig{
			{ID: "feature1", Name: "Feature 1", Tier: "basic", Intercept: InterceptConfig{Package: "test", Function: "F1"}},
			{ID: "feature2", Name: "Feature 2", Tier: "pro", Intercept: InterceptConfig{Package: "test", Function: "F2"}},
		},
	}

	tests := []struct {
		name      string
		featureID string
		want      bool
	}{
		{"existing feature", "feature1", true},
		{"non-existing feature", "feature3", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := manifest.FindFeature(tt.featureID)
			if (got != nil) != tt.want {
				t.Errorf("FindFeature() = %v, want %v", got != nil, tt.want)
			}
		})
	}
}

func TestManifest_GetFeaturesByTier(t *testing.T) {
	manifest := &Manifest{
		SDK: SDKConfig{
			LCCURL:         "http://localhost:7086",
			ProductID:      "test",
			ProductVersion: "1.0.0",
		},
		Features: []FeatureConfig{
			{ID: "f1", Name: "F1", Tier: "basic", Intercept: InterceptConfig{Package: "test", Function: "F1"}},
			{ID: "f2", Name: "F2", Tier: "basic", Intercept: InterceptConfig{Package: "test", Function: "F2"}},
			{ID: "f3", Name: "F3", Tier: "pro", Intercept: InterceptConfig{Package: "test", Function: "F3"}},
		},
	}

	basicFeatures := manifest.GetFeaturesByTier("basic")
	if len(basicFeatures) != 2 {
		t.Errorf("GetFeaturesByTier(basic) = %d, want 2", len(basicFeatures))
	}

	proFeatures := manifest.GetFeaturesByTier("pro")
	if len(proFeatures) != 1 {
		t.Errorf("GetFeaturesByTier(pro) = %d, want 1", len(proFeatures))
	}
}

func TestManifest_GetRequiredFeatures(t *testing.T) {
	manifest := &Manifest{
		SDK: SDKConfig{
			LCCURL:         "http://localhost:7086",
			ProductID:      "test",
			ProductVersion: "1.0.0",
		},
		Features: []FeatureConfig{
			{ID: "f1", Name: "F1", Required: true, Intercept: InterceptConfig{Package: "test", Function: "F1"}},
			{ID: "f2", Name: "F2", Required: false, Intercept: InterceptConfig{Package: "test", Function: "F2"}},
			{ID: "f3", Name: "F3", Required: true, Intercept: InterceptConfig{Package: "test", Function: "F3"}},
		},
	}

	required := manifest.GetRequiredFeatures()
	if len(required) != 2 {
		t.Errorf("GetRequiredFeatures() = %d, want 2", len(required))
	}
}

func TestValidationErrors(t *testing.T) {
	tests := []struct {
		name     string
		manifest *Manifest
		wantErr  bool
	}{
		{
			name: "missing feature name",
			manifest: &Manifest{
				SDK: SDKConfig{
					LCCURL:         "http://localhost:7086",
					ProductID:      "test",
					ProductVersion: "1.0.0",
				},
				Features: []FeatureConfig{
					{
						ID:   "feature1",
						Name: "", // Missing
						Intercept: InterceptConfig{
							Package:  "test",
							Function: "Func",
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "missing intercept function",
			manifest: &Manifest{
				SDK: SDKConfig{
					LCCURL:         "http://localhost:7086",
					ProductID:      "test",
					ProductVersion: "1.0.0",
				},
				Features: []FeatureConfig{
					{
						ID:   "feature1",
						Name: "Feature 1",
						Intercept: InterceptConfig{
							Package: "test",
							// Missing function
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid quota period",
			manifest: &Manifest{
				SDK: SDKConfig{
					LCCURL:         "http://localhost:7086",
					ProductID:      "test",
					ProductVersion: "1.0.0",
				},
				Features: []FeatureConfig{
					{
						ID:   "feature1",
						Name: "Feature 1",
						Intercept: InterceptConfig{
							Package:  "test",
							Function: "Func",
						},
						Quota: &QuotaConfig{
							Limit:  1000,
							Period: "invalid", // Invalid period
						},
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.manifest.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
