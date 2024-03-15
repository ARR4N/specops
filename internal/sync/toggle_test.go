package sync

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/sync/errgroup"
)

func TestToggle(t *testing.T) {
	ctx := context.Background()
	tog := new(Toggle)

	tog.Set(true)
	t.Run("late Wait()", func(t *testing.T) {
		// Wait()ing when the Toggle is "on" MUST NOT block, even if Wait() was
		// called late.
		if err := tog.Wait(ctx); err != nil {
			t.Errorf("%T.Wait(ctx) error %v", tog, err)
		}
	})

	t.Run("idempotent Set doesn't block", func(t *testing.T) {
		for _, set := range []bool{true, false, true} {
			for i := 0; i < 10; i++ {
				tog.Set(set)
			}
		}
	})

	tog.Set(false)
	// All Wait()ing go routines MUST only unblock when Set(true) is called, but
	// no sooner.
	group, gCtx := errgroup.WithContext(ctx)
	unblocked := new(uint64)
	for i := 0; i < 10; i++ {
		group.Go(func() error {
			if err := tog.Wait(gCtx); err != nil {
				return err
			}
			atomic.AddUint64(unblocked, 1)
			return nil
		})
	}

	t.Run("blocks", func(t *testing.T) {
		const timeout = 5 * time.Second
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		if got, want := tog.Wait(ctx), context.DeadlineExceeded; got != want {
			t.Errorf("%T.Wait([ctx with deadline]) got %v; want %v", tog, got, want)
		}
		if n := atomic.LoadUint64(unblocked); n > 0 {
			t.Fatalf("%d go routines unblocked", n)
		}
	})

	t.Run("unblocks", func(t *testing.T) {
		t.Parallel()
		if err := group.Wait(); err != nil {
			t.Errorf("%T.Wait(ctx) error %v", tog, err)
		}
		tog.Close()
	})

	t.Run("Set(true)", func(t *testing.T) {
		t.Parallel()
		tog.Set(true)
	})
}

func TestToggleClose(t *testing.T) {
	ctx := context.Background()
	tog := new(Toggle)

	t.Run("unblock", func(t *testing.T) {
		t.Parallel()
		if got, want := tog.Wait(ctx), ErrToggleClosed; got != want {
			t.Errorf("%T.Wait() got %v; want %v", tog, got, want)
		}
	})

	t.Run("Close()", func(t *testing.T) {
		t.Parallel()
		tog.Close()
	})
}
