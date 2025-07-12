package unstruct

import (
	"context"

	"golang.org/x/sync/errgroup"
)

// DefaultRunner returns the default implementation backed by errgroup.Group.
func DefaultRunner(ctx context.Context) Runner {
	return newErrGroupRunner(ctx)
}

// errGroupRunner is the default implementation backed by errgroup.Group.
type errGroupRunner struct {
	ctx context.Context // derived ctx shared by all tasks
	eg  *errgroup.Group
}

func newErrGroupRunner(parent context.Context) *errGroupRunner {
	eg, ctx := errgroup.WithContext(parent)
	return &errGroupRunner{ctx: ctx, eg: eg}
}

func (r *errGroupRunner) Go(fn func() error) { r.eg.Go(fn) }
func (r *errGroupRunner) Wait() error        { return r.eg.Wait() }
