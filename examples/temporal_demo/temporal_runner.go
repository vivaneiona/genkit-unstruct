// Package temporal_demo provides Temporal workflow integration for the unstruct library.
//
// The TemporalRunner adapts unstruct's concurrent execution model to work seamlessly
// within Temporal's deterministic workflow environment, ensuring reliable and
// fault-tolerant document processing at scale.
package temporal_demo

import (
	"go.temporal.io/sdk/workflow"
)

// TemporalRunner implements the unstruct.Runner interface for Temporal workflows.
//
// This adapter enables unstruct's parallel processing capabilities within Temporal's
// deterministic execution model. Temporal workflows use deterministic coroutines
// rather than OS threads, making traditional synchronization primitives unnecessary.
//
// Key Features:
//   - Deterministic execution: Maintains workflow replay consistency
//   - Error collection: Aggregates errors from async operations
//   - Coroutine coordination: Uses workflow channels for synchronization
//   - Fault tolerance: Survives worker restarts and network failures
type TemporalRunner struct {
	ctx    workflow.Context // Temporal workflow context for deterministic execution
	errors workflow.Channel // Channel to collect errors from async operations
	done   workflow.Channel // Channel to signal completion
	count  int              // Number of pending operations
}

// NewTemporalRunner creates a Temporal-compatible runner for unstruct operations.
//
// The runner uses workflow channels for deterministic coordination of async
// operations, ensuring reliable replay behavior and fault tolerance.
//
// Parameters:
//   - ctx: Temporal workflow context providing deterministic execution guarantees
//
// Returns:
//   - *TemporalRunner: Configured runner ready for unstruct integration
func NewTemporalRunner(ctx workflow.Context) *TemporalRunner {
	logger := workflow.GetLogger(ctx)
	logger.Info("Creating new TemporalRunner",
		"features", "deterministic execution, error aggregation, workflow channels",
	)

	return &TemporalRunner{
		ctx:    ctx,
		errors: workflow.NewBufferedChannel(ctx, 1), // Buffered to prevent blocking
		done:   workflow.NewBufferedChannel(ctx, 1), // Buffered to prevent blocking
		count:  0,
	}
}

// Go schedules a function for asynchronous execution within the Temporal workflow.
//
// This method adapts unstruct's concurrent model to Temporal's deterministic
// execution environment. Each function executes in a workflow coroutine,
// maintaining deterministic replay while enabling parallel LLM API calls.
//
// ⚠️  CRITICAL: The provided function MUST NOT make external API calls directly!
// External calls (LLM APIs, file I/O, network requests) MUST be wrapped in Activities.
// Violating this will break Temporal's determinism and cause replay failures.
//
// Example of CORRECT usage:
//
//	runner.Go(func() error {
//	    // This should execute an Activity, not call external APIs directly
//	    return workflow.ExecuteActivity(ctx, MyLLMActivity, prompt).Get(ctx, &result)
//	})
//
// The implementation:
//  1. Uses workflow.Go for deterministic async execution
//  2. Sends errors to a workflow channel for collection
//  3. Signals completion via workflow channels
//  4. Maintains operation count for proper synchronization
//
// Parameters:
//   - fn: Function to execute asynchronously (MUST only call Activities for external calls)
func (r *TemporalRunner) Go(fn func() error) {
	// Capture operation ID before incrementing to ensure deterministic logging
	opID := r.count + 1
	r.count++

	logger := workflow.GetLogger(r.ctx)
	logger.Info("Scheduling function for async execution",
		"operation_id", opID,
		"total_pending", r.count,
	)

	// Execute function in Temporal workflow coroutine
	workflow.Go(r.ctx, func(ctx workflow.Context) {
		// Create operation-specific logger with immutable ID
		opLogger := workflow.GetLogger(ctx)
		opLogger.Info("Starting async operation execution",
			"operation_id", opID,
		)

		// Execute the provided function
		err := fn()

		// Send error if occurred, otherwise signal completion
		if err != nil {
			opLogger.Error("Async operation failed",
				"operation_id", opID,
				"error", err.Error(),
			)
			r.errors.Send(ctx, err) // Buffered channel, non-blocking
			return
		}

		opLogger.Info("Async operation completed successfully",
			"operation_id", opID,
		)
		r.done.Send(ctx, struct{}{}) // Buffered channel, non-blocking
	})
}

// Wait blocks until all scheduled operations complete and returns the first error.
//
// This method uses workflow.Selector to wait for all operations deterministically,
// ensuring proper synchronization in Temporal's replay-consistent environment.
//
// Behavior:
//   - Waits for all workflow.Go coroutines to complete
//   - Returns the first error encountered (fail-fast semantics)
//   - Returns nil if all operations succeeded
//   - Uses workflow channels for deterministic coordination
//
// Returns:
//   - error: First error encountered, or nil if all operations succeeded
func (r *TemporalRunner) Wait() error {
	logger := workflow.GetLogger(r.ctx)
	logger.Info("Starting to wait for async operations",
		"total_operations", r.count,
	)

	completed := 0
	var firstErr error

	// Use workflow.Selector for deterministic channel operations
	for completed < r.count {
		selector := workflow.NewSelector(r.ctx)

		// Handle error case - capture error and return immediately (fail-fast)
		selector.AddReceive(r.errors, func(c workflow.ReceiveChannel, more bool) {
			c.Receive(r.ctx, &firstErr)
			if firstErr != nil {
				logger.Error("Received error from async operation",
					"error", firstErr.Error(),
					"completed_so_far", completed,
					"total_operations", r.count,
				)
			}
		})

		// Handle completion case - increment counter
		selector.AddReceive(r.done, func(c workflow.ReceiveChannel, more bool) {
			var signal struct{}
			c.Receive(r.ctx, &signal)
			completed++
			logger.Info("Async operation completed",
				"completed_operations", completed,
				"total_operations", r.count,
				"remaining", r.count-completed,
			)
		})

		selector.Select(r.ctx)

		// Return immediately on first error (fail-fast semantics)
		if firstErr != nil {
			logger.Error("Returning first error encountered",
				"error", firstErr.Error(),
				"completed_operations", completed,
				"total_operations", r.count,
			)
			return firstErr
		}
	}

	logger.Info("All async operations completed successfully",
		"total_operations", r.count,
	)
	return nil
}
