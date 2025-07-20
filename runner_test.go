package unstruct

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultRunner(t *testing.T) {
	ctx := context.Background()
	runner := DefaultRunner(ctx)

	require.NotNil(t, runner, "DefaultRunner returned nil")

	// Verify it implements the Runner interface
	var _ Runner = runner

	// Verify it's the expected concrete type
	_, ok := runner.(*errGroupRunner)
	assert.True(t, ok, "DefaultRunner should return *errGroupRunner, got %T", runner)
}

func TestErrGroupRunner_Go_Success(t *testing.T) {
	ctx := context.Background()
	runner := DefaultRunner(ctx)

	var counter int32
	var wg sync.WaitGroup

	// Start multiple goroutines
	for i := 0; i < 5; i++ {
		wg.Add(1)
		runner.Go(func() error {
			defer wg.Done()
			atomic.AddInt32(&counter, 1)
			return nil
		})
	}

	// Wait for all to complete
	err := runner.Wait()
	assert.NoError(t, err, "Expected no error, got %v", err)

	// Ensure all goroutines ran
	wg.Wait()
	assert.Equal(t, int32(5), atomic.LoadInt32(&counter), "Expected counter to be 5, got %d", atomic.LoadInt32(&counter))
}

func TestErrGroupRunner_Go_WithError(t *testing.T) {
	ctx := context.Background()
	runner := DefaultRunner(ctx)

	expectedErr := errors.New("test error")

	// Start one goroutine that succeeds and one that fails
	runner.Go(func() error {
		time.Sleep(10 * time.Millisecond)
		return nil
	})

	runner.Go(func() error {
		return expectedErr
	})

	// Wait should return the error
	err := runner.Wait()
	assert.Error(t, err, "Expected error, got nil")
	assert.Equal(t, expectedErr, err, "Expected %v, got %v", expectedErr, err)
}

func TestErrGroupRunner_Go_MultipleErrors(t *testing.T) {
	ctx := context.Background()
	runner := DefaultRunner(ctx)

	err1 := errors.New("error 1")
	err2 := errors.New("error 2")

	// Start multiple goroutines that fail
	runner.Go(func() error {
		return err1
	})

	runner.Go(func() error {
		return err2
	})

	// Wait should return one of the errors (errgroup returns the first)
	err := runner.Wait()
	assert.Error(t, err, "Expected error, got nil")
	// Could be either error depending on timing
	assert.True(t, err == err1 || err == err2, "Expected error1 or error2, got %v", err)
}

func TestErrGroupRunner_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	runner := DefaultRunner(ctx)

	// Start a long-running goroutine
	runner.Go(func() error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(1 * time.Second):
			return nil
		}
	})

	// Cancel the context
	cancel()

	// Wait should return context.Canceled
	err := runner.Wait()
	assert.True(t, errors.Is(err, context.Canceled), "Expected context.Canceled, got %v", err)
}

func TestErrGroupRunner_ContextTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	runner := DefaultRunner(ctx)

	// Start a goroutine that takes longer than the timeout
	runner.Go(func() error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(200 * time.Millisecond):
			return nil
		}
	})

	// Wait should return context.DeadlineExceeded
	err := runner.Wait()
	assert.True(t, errors.Is(err, context.DeadlineExceeded), "Expected context.DeadlineExceeded, got %v", err)
}

func TestErrGroupRunner_EmptyRunner(t *testing.T) {
	ctx := context.Background()
	runner := DefaultRunner(ctx)

	// Call Wait without scheduling any work
	err := runner.Wait()
	assert.NoError(t, err, "Expected no error for empty runner, got %v", err)
}

func TestErrGroupRunner_ConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	runner := DefaultRunner(ctx)

	var counter int32
	numGoroutines := 100

	// Start many goroutines concurrently
	for i := 0; i < numGoroutines; i++ {
		runner.Go(func() error {
			atomic.AddInt32(&counter, 1)
			// Simulate some work
			time.Sleep(time.Millisecond)
			return nil
		})
	}

	err := runner.Wait()
	assert.NoError(t, err, "Expected no error, got %v", err)

	assert.Equal(t, int32(numGoroutines), atomic.LoadInt32(&counter), "Expected counter to be %d, got %d", numGoroutines, atomic.LoadInt32(&counter))
}

func TestNewErrGroupRunner(t *testing.T) {
	ctx := context.Background()
	runner := newErrGroupRunner(ctx, 5) // Add concurrency limit

	require.NotNil(t, runner, "newErrGroupRunner returned nil")
	require.NotNil(t, runner.ctx, "runner.ctx should not be nil")
	require.NotNil(t, runner.eg, "runner.eg should not be nil")
	require.NotNil(t, runner.sem, "runner.sem should not be nil")

	// Check that semaphore has correct capacity
	assert.Equal(t, 5, cap(runner.sem), "semaphore should have capacity of 5")

	// The context should be derived from the parent
	assert.NotEqual(t, ctx, runner.ctx, "runner.ctx should be a derived context, not the same as parent")
}

// TestRunnerInterface ensures the interface is properly implemented
func TestRunnerInterface(t *testing.T) {
	ctx := context.Background()

	// Test that DefaultRunner returns something that implements Runner
	var runner Runner = DefaultRunner(ctx)

	// Test the interface methods exist and can be called
	runner.Go(func() error { return nil })
	err := runner.Wait()

	assert.NoError(t, err, "Basic interface test failed: %v", err)
}

// BenchmarkRunner tests performance characteristics
func BenchmarkErrGroupRunner(b *testing.B) {
	ctx := context.Background()

	b.Run("Sequential", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			runner := DefaultRunner(ctx)
			runner.Go(func() error { return nil })
			_ = runner.Wait()
		}
	})

	b.Run("Concurrent", func(b *testing.B) {
		runner := DefaultRunner(ctx)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			runner.Go(func() error { return nil })
		}
		_ = runner.Wait()
	})
}
