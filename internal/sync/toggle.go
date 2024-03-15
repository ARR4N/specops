package sync

import (
	"context"
	"errors"
	"sync"
)

// A Toggle allows for Wait()ing until a condition is true. Unlike a broadcast
// mechanism that may be missed if waiting begins after a signal, waiting on an
// already "on" Toggle returns immediately.
//
// The zero value for a Toggle is equivalent to Set(false). A Toggle MUST NOT be
// copied as it contains a sync.Mutex.
//
// Toggle.Set(true) is a replacement for sync.Cond.Broadcast().
//
// The implementation uses a channel with a single-item buffer. When Set() to
// true, the Toggle adds an item to the buffer, and when Set() to false, it
// removes said item. All calls to Wait() receive on the channel to unblock,
// and then immediately return the item. This allows for Context cancellation to
// be honoured.
type Toggle struct {
	mu    sync.Mutex
	state bool

	// MUST NOT be accessed directly. Use sigChan() or
	// sigChanWhenAlreadyLocked().
	signal chan struct{}
}

// sigChan locks t and returns t.sigChanWhenAlreadyLocked().
func (t *Toggle) sigChan() chan struct{} {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.sigChanWhenAlreadyLocked()
}

// sigChanWhenAlreadyLocked returns t.signal, make()ing it if nil.
func (t *Toggle) sigChanWhenAlreadyLocked() chan struct{} {
	if t.signal == nil {
		t.signal = make(chan struct{}, 1)
	}
	return t.signal
}

// Close closes the Toggle. All Wait()ers, current and future, unblock and
// return ErrToggleClosed.
func (t *Toggle) Close() {
	close(t.sigChan())
}

// ErrToggleClosed is returned by Toggle.Wait() if Toggle.Close() was called.
var ErrToggleClosed = errors.New("toggle closed")

// Wait blocks until the Toggle is Set() to true. If the last call to Set() was
// Set(true) then Wait unblocks immediately.
func (t *Toggle) Wait(ctx context.Context) error {
	ch := t.sigChan()

	select {
	case <-ctx.Done():
		return ctx.Err()

	case x, ok := <-ch:
		if !ok {
			return ErrToggleClosed
		}

		ch <- x
		return nil
	}
}

// Set sets the state of the Toggle. If the state is true, all current and
// future calls to Wait() will unblock. Calls to Set are idempotent.
//
// Behaviour of Set() is undefined on a Close()d Toggle.
func (t *Toggle) Set(state bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if state == t.state {
		return
	}
	t.state = state

	ch := t.sigChanWhenAlreadyLocked()
	if state {
		ch <- struct{}{}
	} else {
		<-ch
	}
}

// State returns the last value sent to Set(), or false if Set() is yet to be
// called.
func (t *Toggle) State() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.state
}
