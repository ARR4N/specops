package runopts_test

import (
	"bytes"
	"testing"

	. "github.com/solidifylabs/specialops"
)

func TestDebugger(t *testing.T) {
	const retVal = 42
	code := Code{
		PUSH0, PUSH(1), PUSH(2),
		Fn(MSTORE, PUSH(0), PUSH(retVal)),
		Fn(RETURN, PUSH0, PUSH(32)),
	}

	dbg, results := code.StartDebugging(nil)
	state := dbg.State() // can be called any time

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

	var want [32]byte
	want[31] = retVal

	got, err := results()
	if err != nil || !bytes.Equal(got, want[:]) {
		t.Errorf("%T.StartDebugging() results function returned %#x, err = %v; want %#x; nil error", code, got, err, want[:])
	}
}
