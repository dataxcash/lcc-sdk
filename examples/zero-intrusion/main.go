package main

import (
	"context"
	"fmt"
	"log"

	"github.com/yourorg/lcc-sdk/pkg/client"
	"github.com/yourorg/lcc-sdk/pkg/config"
)

func main() {
	// Load configuration
	cfg := &config.SDKConfig{
		LCCURL:         "http://localhost:7086",
		ProductID:      "demo-app",
		ProductVersion: "2.0.0",
	}

	// Create LCC client
	lccClient, err := client.NewClient(cfg)
	if err != nil {
		log.Fatalf("Failed to create LCC client: %v", err)
	}
	defer lccClient.Close()

	// Register helper functions for zero-intrusion API
	helpers := &client.HelperFunctions{
		// QuotaConsumer: Calculate consumption based on batch size
		QuotaConsumer: func(ctx context.Context, args ...interface{}) int {
			if len(args) > 0 {
				if batchSize, ok := args[0].(int); ok {
					return batchSize
				}
			}
			return 1
		},

		// TPSProvider: Provide current TPS (optional, SDK tracks internally if not provided)
		TPSProvider: func() float64 {
			return getCurrentTPS()
		},

		// CapacityCounter: Count active users (required for capacity limits)
		CapacityCounter: func() int {
			return countActiveUsers()
		},
	}

	if err := lccClient.RegisterHelpers(helpers); err != nil {
		log.Fatalf("Failed to register helpers: %v", err)
	}

	// Register with LCC server
	if err := lccClient.Register(); err != nil {
		log.Fatalf("Failed to register with LCC: %v", err)
	}

	// Example 1: Simple quota consumption
	if err := exampleSimpleConsume(lccClient); err != nil {
		log.Printf("Example 1 failed: %v", err)
	}

	// Example 2: Quota consumption with helper function
	if err := exampleConsumeWithHelper(lccClient, 10); err != nil {
		log.Printf("Example 2 failed: %v", err)
	}

	// Example 3: TPS check
	if err := exampleCheckTPS(lccClient); err != nil {
		log.Printf("Example 3 failed: %v", err)
	}

	// Example 4: Capacity check
	if err := exampleCheckCapacity(lccClient); err != nil {
		log.Printf("Example 4 failed: %v", err)
	}

	// Example 5: Concurrency control
	if err := exampleConcurrency(lccClient); err != nil {
		log.Printf("Example 5 failed: %v", err)
	}

	fmt.Println("All examples completed successfully!")
}

// Example 1: Simple quota consumption
func exampleSimpleConsume(lccClient *client.Client) error {
	allowed, remaining, err := lccClient.Consume(1)
	if err != nil {
		return fmt.Errorf("consume failed: %w", err)
	}

	if !allowed {
		return fmt.Errorf("quota exceeded: remaining=%d", remaining)
	}

	fmt.Printf("✅ Example 1: Consumed 1 unit, %d remaining\n", remaining)
	return nil
}

// Example 2: Quota consumption with helper function
func exampleConsumeWithHelper(lccClient *client.Client, batchSize int) error {
	ctx := context.Background()

	// The helper function will calculate consumption based on batchSize
	allowed, remaining, err := lccClient.ConsumeWithContext(ctx, batchSize)
	if err != nil {
		return fmt.Errorf("consume with context failed: %w", err)
	}

	if !allowed {
		return fmt.Errorf("quota exceeded: remaining=%d", remaining)
	}

	fmt.Printf("✅ Example 2: Consumed %d units, %d remaining\n", batchSize, remaining)
	return nil
}

// Example 3: TPS check (SDK tracks internally or uses TPSProvider)
func exampleCheckTPS(lccClient *client.Client) error {
	allowed, maxTPS, err := lccClient.CheckTPS()
	if err != nil {
		return fmt.Errorf("TPS check failed: %w", err)
	}

	if !allowed {
		return fmt.Errorf("TPS exceeded: max=%.2f", maxTPS)
	}

	fmt.Printf("✅ Example 3: TPS check passed (max=%.2f)\n", maxTPS)
	return nil
}

// Example 4: Capacity check using helper function
func exampleCheckCapacity(lccClient *client.Client) error {
	// Automatically calls CapacityCounter helper
	allowed, maxCapacity, err := lccClient.CheckCapacityWithHelper()
	if err != nil {
		return fmt.Errorf("capacity check failed: %w", err)
	}

	if !allowed {
		return fmt.Errorf("capacity exceeded: max=%d", maxCapacity)
	}

	fmt.Printf("✅ Example 4: Capacity check passed (max=%d)\n", maxCapacity)
	return nil
}

// Example 5: Concurrency control
func exampleConcurrency(lccClient *client.Client) error {
	// Acquire a concurrency slot
	release, allowed, err := lccClient.AcquireSlot()
	if err != nil {
		return fmt.Errorf("acquire slot failed: %w", err)
	}

	if !allowed {
		return fmt.Errorf("concurrency limit exceeded")
	}

	// IMPORTANT: Must release slot when done
	defer release()

	// Perform concurrent operation
	fmt.Println("✅ Example 5: Slot acquired, performing operation...")

	// Simulate work
	// ... do work ...

	return nil
}

// Mock helper functions (replace with real implementations)

func getCurrentTPS() float64 {
	// In real application, return actual TPS measurement
	return 50.0
}

func countActiveUsers() int {
	// In real application, count from database or cache
	return 100
}
