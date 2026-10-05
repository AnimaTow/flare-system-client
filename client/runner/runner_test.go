package runner

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

type runFunc func(ctx context.Context) error

func (f runFunc) Run(ctx context.Context) error { return f(ctx) }

// main exits 1 unless the cause is its own shutdown signal.
func TestRunAsyncCause(t *testing.T) {
	errFailed := errors.New("failed")
	errSignal := errors.New("signal")

	t.Run("runner error becomes the cause", func(t *testing.T) {
		ctx, cancel := context.WithCancelCause(t.Context())
		var wg sync.WaitGroup
		RunAsync(ctx, cancel, &wg, runFunc(func(context.Context) error { return errFailed }))
		RunAsync(ctx, cancel, &wg, runFunc(func(ctx context.Context) error { <-ctx.Done(); return ctx.Err() }))
		wg.Wait()

		require.ErrorIs(t, context.Cause(ctx), errFailed)
	})

	t.Run("errors while draining keep the signal cause", func(t *testing.T) {
		ctx, cancel := context.WithCancelCause(t.Context())
		var wg sync.WaitGroup
		RunAsync(ctx, cancel, &wg, runFunc(func(ctx context.Context) error { <-ctx.Done(); return errFailed }))
		cancel(errSignal)
		wg.Wait()

		require.ErrorIs(t, context.Cause(ctx), errSignal)
	})

	t.Run("nil return does not cancel", func(t *testing.T) {
		ctx, cancel := context.WithCancelCause(t.Context())
		defer cancel(nil)
		var wg sync.WaitGroup
		RunAsync(ctx, cancel, &wg, runFunc(func(context.Context) error { return nil }))
		wg.Wait()

		require.NoError(t, context.Cause(ctx))
	})
}
