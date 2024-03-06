package runopts_test

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/solidifylabs/specops/stack"

	. "github.com/solidifylabs/specops"
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
			dbg, results, err := code.StartDebugging(nil)
			if err != nil { // compilation error
				t.Fatalf("%T.StartDebugging(nil) error %v", code, err)
			}
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

func TestDebuggerCompilationError(t *testing.T) {
	code := Code{
		stack.ExpectDepth(5),
	}
	if _, _, err := code.StartDebugging(nil); err == nil || !strings.Contains(err.Error(), "Compile()") {
		t.Errorf("%T.StartDebugging(nil) with known compilation failure got err %v; want containing %q", code, err, "Compile()")
	}
}

func TestDebuggerErrors(t *testing.T) {
	tests := []struct {
		name        string
		code        Code
		wantErrType reflect.Type // errors.As doesn't play nicely with any/error
	}{
		{
			name: "immediate underflow",
			code: Code{
				stack.SetDepth(2), RETURN, // compiles to {RETURN}
			},
			wantErrType: reflect.TypeOf(new(vm.ErrStackUnderflow)),
		},
		{
			name: "delayed underflow",
			code: Code{
				PUSH0, stack.SetDepth(2), RETURN, // compiles to {PUSH0, RETURN}
			},
			wantErrType: reflect.TypeOf(new(vm.ErrStackUnderflow)),
		},
		{
			name: "invalid opcode",
			code: Code{
				Raw{byte(INVALID)},
			},
			wantErrType: reflect.TypeOf(new(vm.ErrInvalidOpCode)),
		},
		{
			name: "explicit revert",
			code: Code{
				Fn(REVERT, PUSH0, PUSH0),
			},
			wantErrType: reflect.TypeOf(errors.New("")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dbg, results, err := tt.code.StartDebugging(nil)
			if err != nil {
				t.Fatalf("%T.StartDebugging(nil) error %v", tt.code, err)
			}
			dbg.FastForward()

			if err := dbg.State().Err; reflect.TypeOf(err) != tt.wantErrType {
				t.Errorf("%T.State().Err = %T(%v); want type %v", dbg, err, err, tt.wantErrType)
			}
			if _, err := results(); reflect.TypeOf(errors.Unwrap(err)) != tt.wantErrType {
				t.Errorf("%T.StartDebugging() results function returned error %T(%v); want type %v wrapped", dbg, err, err, tt.wantErrType)
			}
		})
	}
}
