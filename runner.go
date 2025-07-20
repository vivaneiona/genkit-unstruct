package unstruct

import (
	"context"
	"runtime"

	"golang.org/x/sync/errgroup"
)

// DefaultRunner returns the default implementation backed by errgroup.Group.
func DefaultRunner(ctx context.Context) Runner {
	return newErrGroupRunner(ctx, runtime.NumCPU())
}

// NewLimitedRunner creates a runner with bounded concurrency.
func NewLimitedRunner(ctx context.Context, maxConcurrency int) Runner {
	return newErrGroupRunner(ctx, maxConcurrency)
}

// errGroupRunner is the default implementation backed by errgroup.Group.
type errGroupRunner struct {
	ctx context.Context // derived ctx shared by all tasks
	eg  *errgroup.Group
	sem chan struct{} // concurrency gate
}

func newErrGroupRunner(parent context.Context, maxConcurrency int) *errGroupRunner {
	eg, ctx := errgroup.WithContext(parent)
	return &errGroupRunner{
		ctx: ctx,
		eg:  eg,
		sem: make(chan struct{}, maxConcurrency),
	}
}

func (r *errGroupRunner) Go(fn func() error) {
	r.eg.Go(func() error {
		r.sem <- struct{}{}        // acquire
		defer func() { <-r.sem }() // release
		return fn()
	})
}

func (r *errGroupRunner) Wait() error { return r.eg.Wait() }
