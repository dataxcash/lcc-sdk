package client

import (
	"context"
	"fmt"
)

// HelperFunctions contains optional/required helper functions for customizing
// limit enforcement behavior. These helpers enable zero-intrusion integration
// by allowing applications to provide custom logic without modifying business code.
//
// Usage:
//   - QuotaConsumer: Calculate custom consumption amounts based on function arguments
//   - TPSProvider: Provide current TPS measurement (optional, SDK tracks internally)
//   - CapacityCounter: Count current resource usage (REQUIRED for capacity limits)
type HelperFunctions struct {
	// QuotaConsumer (Optional): Calculate custom consumption amount
	// If not provided, defaults to consuming 1 unit per call
	// Args: function parameters from intercepted call
	// Returns: amount of quota units to consume
	//
	// Example:
	//   QuotaConsumer: func(ctx context.Context, args ...interface{}) int {
	//       if len(args) > 0 {
	//           if batchSize, ok := args[0].(int); ok {
	//               return batchSize
	//           }
	//       }
	//       return 1
	//   }
	QuotaConsumer func(ctx context.Context, args ...interface{}) int

	// TPSProvider (Optional): Provide current TPS measurement
	// If not provided, SDK auto-tracks TPS internally
	// Returns: current transactions per second
	//
	// Example:
	//   TPSProvider: func() float64 {
	//       return myMetrics.GetCurrentTPS()
	//   }
	TPSProvider func() float64

	// CapacityCounter (Required): Count current resource usage
	// MUST be provided for capacity limits to work
	// Returns: current count of persistent resources
	//
	// Example:
	//   CapacityCounter: func() int {
	//       return database.CountActiveUsers()
	//   }
	CapacityCounter func() int
}

// Validate validates the helper functions configuration
// Returns error if required helpers are missing or invalid
func (h *HelperFunctions) Validate() error {
	if h == nil {
		return fmt.Errorf("HelperFunctions cannot be nil")
	}

	// CapacityCounter is required for capacity limits
	// If you don't use capacity limits, you can provide a stub function
	if h.CapacityCounter == nil {
		return fmt.Errorf("CapacityCounter is required (provide stub if not using capacity limits)")
	}

	return nil
}

// SetDefaults sets default implementations for optional helpers
func (h *HelperFunctions) SetDefaults(client *Client) {
	if h.QuotaConsumer == nil {
		h.QuotaConsumer = defaultQuotaConsumer
	}

	if h.TPSProvider == nil {
		h.TPSProvider = func() float64 {
			return client.getInternalTPS()
		}
	}
}

// defaultQuotaConsumer is the default implementation that consumes 1 unit
func defaultQuotaConsumer(ctx context.Context, args ...interface{}) int {
	return 1
}
