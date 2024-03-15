package evmdebug_test

import (
	"runtime"
	"testing"

	. "github.com/solidifylabs/specops"
)

func TestStepSynchronisation(t *testing.T) {
	var code Code
	const n = 1_000
	for i := 0; i < n; i++ {
		// An operation that is likely to take longer than the return of
		// dbg.Step() if no synchronisation is in place.
		code = append(code, Fn(KECCAK256, PUSH0, PUSH(4096)))
	}

	// Synchronise the start of parallel tests to maximise load and increase
	// probability of checking the stack before the KECCAK256 is finished (if
	// debugger syncing is broken).
	start := make(chan struct{})

	for tt := 0; tt < runtime.GOMAXPROCS(0)*2; tt++ {
		t.Run("", func(t *testing.T) {
			t.Parallel()

			<-start

			dbg, _, err := code.StartDebugging(nil)
			if err != nil {
				t.Fatalf("%T.StartDebugging(nil) error %v", code, err)
			}

			state := dbg.State()
			for i := 0; i < n; i++ {
				dbg.Step()
				dbg.Step()
				dbg.Step()
				if got, want := len(state.ScopeContext.Stack.Data()), i+1; got != want {
					t.Fatalf("After %dº run of SHA3; got stack depth %d; want %d", i+1, got, want)
				}
			}
		})
	}

	close(start)
}
