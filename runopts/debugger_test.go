package runopts_test

import (
	"testing"

	. "github.com/solidifylabs/specialops"
	. "github.com/solidifylabs/specialops/runopts" // Note that we're in the runopts_test package to avoid circular deps
)

func TestDebugger(t *testing.T) {
	c := Code{
		PUSH0, PUSH(1), PUSH(2),
		Fn(MSTORE, PUSH(0), PUSH(42)),
		Fn(RETURN, PUSH0, PUSH(32)),
	}

	dbg := NewDebugger()

	done := make(chan struct{})
	go func() {
		defer func() { close(done) }()
		if _, err := c.Run(nil, dbg); err != nil {
			t.Errorf("%T.Run(nil, %T) error %v", c, dbg, err)
		}
	}()

	state := dbg.State() // can be called any time
	dbg.Wait()

	wantPCs := []uint64{0}
	pcIncrs := []uint64{
		1, // PUSH0
		2, // PUSH1
		2, // PUSH1
		2, // PUSH1
		1, // PUSH0
		1, // MSTORE
		2, // PUSH1
		1, // PUSH0
		// RETURN
	}
	for i, incr := range pcIncrs {
		wantPCs = append(wantPCs, wantPCs[i]+incr)
	}

	for i := uint64(0); !dbg.Done(); i++ {
		t.Run("step", func(t *testing.T) {
			dbg.Step()
			if got, want := state.PC, wantPCs[i]; got != want {
				t.Errorf("%T.State().PC got %d; want %d", dbg, got, want)
			}
			if err := state.Err; err != nil {
				t.Errorf("%T.State().Err got %v; want nil", dbg, err)
			}
		})
	}

	<-done
}
