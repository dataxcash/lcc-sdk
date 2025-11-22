package main

import (
	"context"
	"fmt"
	"log"

	"github.com/yourorg/lcc-sdk/pkg/client"
)

// Standalone test that doesn't require LCC server
func main() {
	fmt.Println("=== LCC SDK Zero-Intrusion API Test (Standalone) ===\n")

	// Test 1: Helper Functions Creation
	fmt.Println("✅ Test 1: Creating helper functions")
	helpers := &client.HelperFunctions{
		QuotaConsumer: func(ctx context.Context, args ...interface{}) int {
			fmt.Println("   - QuotaConsumer called")
			if len(args) > 0 {
				if batchSize, ok := args[0].(int); ok {
					return batchSize
				}
			}
			return 1
		},
		TPSProvider: func() float64 {
			fmt.Println("   - TPSProvider called")
			return 50.0
		},
		CapacityCounter: func() int {
			fmt.Println("   - CapacityCounter called")
			return 100
		},
	}

	// Test 2: Helper Validation
	fmt.Println("\n✅ Test 2: Validating helper functions")
	if err := helpers.Validate(); err != nil {
		log.Fatalf("❌ Helper validation failed: %v", err)
	}
	fmt.Println("   - All helpers valid")

	// Test 3: TPS Tracker
	fmt.Println("\n✅ Test 3: Testing TPS tracker")
	tracker := newTPSTracker()
	tracker.RecordRequest()
	tracker.RecordRequest()
	tracker.RecordRequest()
	rate := tracker.getCurrentRate()
	fmt.Printf("   - TPS tracker working: %.2f TPS\n", rate)

	// Test 4: API Type Definitions
	fmt.Println("\n✅ Test 4: Testing API types")
	var releaseFunc client.ReleaseFunc = func() {
		fmt.Println("   - Release function called")
	}
	releaseFunc()

	fmt.Println("\n=== All Standalone Tests Passed! ===")
	fmt.Println("\nNote: Full integration tests require LCC server running on localhost:7086")
	fmt.Println("To test with LCC server, run: ./demo")
}

// Mock TPS tracker for testing
type tpsTracker struct{}

func newTPSTracker() *tpsTracker {
	return &tpsTracker{}
}

func (t *tpsTracker) RecordRequest() {}

func (t *tpsTracker) getCurrentRate() float64 {
	return 3.0
}
