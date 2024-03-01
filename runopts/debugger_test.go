package runopts_test

import (
	"bytes"
	"fmt"
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

	for ffAt, steps := 0, len(wantPCs); ffAt < steps; ffAt++ { // using range wantPCs, while the same, is misleading
		t.Run(fmt.Sprintf("fast-forward after step %d", ffAt), func(t *testing.T) {
			dbg, results := code.StartDebugging(nil)
			defer dbg.FastForward() // best practice to avoid resource leakage

			state := dbg.State() // can be called any time

			for step := 0; !dbg.Done(); step++ {
				t.Run("step", func(t *testing.T) {
					dbg.Step()
					if got, want := state.PC, wantPCs[step]; got != want {
						t.Errorf("%T.State().PC got %d; want %d", dbg, got, want)
					}
					if err := state.Err; err != nil {
						t.Errorf("%T.State().Err got %v; want nil", dbg, err)
					}
				})

				if step == ffAt {
					dbg.FastForward()
					if !dbg.Done() {
						t.Errorf("%T.Done() after %T.FastForward() got false; want true", dbg, dbg)
					}
				}
			}

			got, err := results()
			var want [32]byte
			want[31] = retVal
			if err != nil || !bytes.Equal(got, want[:]) {
				t.Errorf("%T.StartDebugging() results function returned %#x, err = %v; want %#x; nil error", code, got, err, want[:])
			}
		})
	}
}
