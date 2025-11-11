package tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/yourorg/lcc-sdk/pkg/client"
	"github.com/yourorg/lcc-sdk/pkg/config"
)

func TestSDKIntegration(t *testing.T) {
	// Skip if LCC_URL not set
	lccURL := "https://localhost:8088"
	
	// Create SDK config
	cfg := &config.SDKConfig{
		LCCURL:         lccURL,
		ProductID:      "demo-app",
		ProductVersion: "1.0.0",
		Timeout:        30 * time.Second,
		CacheTTL:       10 * time.Second,
	}

	// Create client
	t.Log("Creating SDK client...")
	c, err := client.NewClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer c.Close()

	t.Logf("Instance ID: %s", c.GetInstanceID())

	// Test 1: Register
	t.Log("Testing registration...")
	if err := c.Register(); err != nil {
		t.Fatalf("Failed to register: %v", err)
	}
	t.Log("✓ Registration successful")

	// Test 2: Check feature (should be enabled for demo-app)
	t.Log("Testing feature check - advanced_analytics...")
	status, err := c.CheckFeature("advanced_analytics")
	if err != nil {
		t.Fatalf("Failed to check feature: %v", err)
	}
	t.Logf("✓ Feature check result: enabled=%v, reason=%s", status.Enabled, status.Reason)

	if !status.Enabled {
		t.Error("Expected advanced_analytics to be enabled for demo-app")
	}

	// Test 3: Check non-existent feature (should be disabled)
	t.Log("Testing feature check - non_existent_feature...")
	status, err = c.CheckFeature("non_existent_feature")
	if err != nil {
		t.Fatalf("Failed to check feature: %v", err)
	}
	t.Logf("✓ Feature check result: enabled=%v, reason=%s", status.Enabled, status.Reason)

	// Test 4: Report usage
	t.Log("Testing usage reporting...")
	if err := c.ReportUsage("advanced_analytics", 1); err != nil {
		t.Fatalf("Failed to report usage: %v", err)
	}
	t.Log("✓ Usage reporting successful")

	// Test 5: Check cache (should return cached result)
	t.Log("Testing cache...")
	start := time.Now()
	status, err = c.CheckFeature("advanced_analytics")
	if err != nil {
		t.Fatalf("Failed to check feature from cache: %v", err)
	}
	elapsed := time.Since(start)
	t.Logf("✓ Cache check took: %v (should be <1ms if cached)", elapsed)

	if elapsed > 10*time.Millisecond {
		t.Log("Warning: Cache might not be working (took >10ms)")
	}

	// Test 6: Clear cache and check again
	t.Log("Testing cache clear...")
	c.ClearCache()
	status, err = c.CheckFeature("advanced_analytics")
	if err != nil {
		t.Fatalf("Failed to check feature after cache clear: %v", err)
	}
	t.Log("✓ Cache clear successful")

	t.Log("\n=== All integration tests passed! ===")
}

func TestSDKMultipleInstances(t *testing.T) {
	lccURL := "https://localhost:8088"
	
	// Create multiple clients to simulate multiple app instances
	numInstances := 3
	clients := make([]*client.Client, numInstances)

	for i := 0; i < numInstances; i++ {
		cfg := &config.SDKConfig{
			LCCURL:         lccURL,
			ProductID:      "demo-app",
			ProductVersion: "1.0.0",
			Timeout:        30 * time.Second,
			CacheTTL:       10 * time.Second,
		}

		c, err := client.NewClient(cfg)
		if err != nil {
			t.Fatalf("Failed to create client %d: %v", i, err)
		}
		defer c.Close()

		clients[i] = c
		t.Logf("Created instance %d with ID: %s", i, c.GetInstanceID())

		// Register each instance
		if err := c.Register(); err != nil {
			t.Fatalf("Failed to register instance %d: %v", i, err)
		}
	}

	// Each instance checks features
	for i, c := range clients {
		status, err := c.CheckFeature("advanced_analytics")
		if err != nil {
			t.Fatalf("Instance %d failed to check feature: %v", i, err)
		}

		if !status.Enabled {
			t.Errorf("Instance %d: expected feature to be enabled", i)
		}

		// Report usage
		if err := c.ReportUsage("advanced_analytics", 1); err != nil {
			t.Fatalf("Instance %d failed to report usage: %v", i, err)
		}

		t.Logf("✓ Instance %d completed all operations", i)
	}

	t.Log("\n=== Multi-instance test passed! ===")
}

// Benchmark feature check performance
func BenchmarkFeatureCheck(b *testing.B) {
	cfg := &config.SDKConfig{
		LCCURL:         "https://localhost:8088",
		ProductID:      "demo-app",
		ProductVersion: "1.0.0",
		Timeout:        30 * time.Second,
		CacheTTL:       10 * time.Second,
	}

	c, err := client.NewClient(cfg)
	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}
	defer c.Close()

	if err := c.Register(); err != nil {
		b.Fatalf("Failed to register: %v", err)
	}

	// Warm up cache
	c.CheckFeature("advanced_analytics")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := c.CheckFeature("advanced_analytics")
		if err != nil {
			b.Fatalf("Feature check failed: %v", err)
		}
	}
}

// Example usage
func ExampleClient() {
	cfg := &config.SDKConfig{
		LCCURL:         "https://localhost:8088",
		ProductID:      "my-app",
		ProductVersion: "1.0.0",
		Timeout:        30 * time.Second,
		CacheTTL:       10 * time.Second,
	}

	c, err := client.NewClient(cfg)
	if err != nil {
		panic(err)
	}
	defer c.Close()

	// Register with LCC
	if err := c.Register(); err != nil {
		panic(err)
	}

	// Check if feature is enabled
	status, err := c.CheckFeature("premium_feature")
	if err != nil {
		panic(err)
	}

	if status.Enabled {
		fmt.Println("Feature is enabled!")
		// Use premium feature
		
		// Report usage
		c.ReportUsage("premium_feature", 1)
	} else {
		fmt.Printf("Feature disabled: %s\n", status.Reason)
	}
}
