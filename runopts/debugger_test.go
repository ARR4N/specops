package runopts_test

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/solidifylabs/specops/runopts"
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
			if err != nil || !bytes.Equal(got.ReturnData, want[:]) {
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
		name          string
		code          Code
		wantVMErrCode int
	}{
		{
			name: "immediate underflow",
			code: Code{
				stack.SetDepth(2), RETURN, // compiles to {RETURN}
			},
			wantVMErrCode: vm.VMErrorCodeStackUnderflow,
		},
		{
			name: "delayed underflow",
			code: Code{
				PUSH0, stack.SetDepth(2), RETURN, // compiles to {PUSH0, RETURN}
			},
			wantVMErrCode: vm.VMErrorCodeStackUnderflow,
		},
		{
			name: "invalid opcode",
			code: Code{
				Raw{byte(INVALID)},
			},
			wantVMErrCode: vm.VMErrorCodeInvalidOpCode,
		},
		{
			name: "explicit revert",
			code: Code{
				Fn(REVERT, PUSH0, PUSH0),
			},
			wantVMErrCode: vm.VMErrorCodeExecutionReverted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dbg, results, err := tt.code.StartDebugging(nil, runopts.NoErrorOnRevert())
			if err != nil {
				t.Fatalf("%T.StartDebugging(nil) error %v", tt.code, err)
			}
			dbg.FastForward()

			errs := []struct {
				name string
				err  error
			}{
				{
					name: fmt.Sprintf("%T.State().Err", dbg),
					err:  dbg.State().Err,
				},
				{
					name: fmt.Sprintf("%T.StartDebugging() -> %T.Err", dbg, &core.ExecutionResult{}),
					err: func() error {
						r, err := results()
						if err != nil {
							// This would mean that the error occurred outside of the code execution.
							t.Fatalf("%T.StartDebugging() results function error %v", dbg, err)
						}
						return vm.VMErrorFromErr(r.Err)
					}(),
				},
			}

			for _, e := range errs {
				t.Run(e.name, func(t *testing.T) {
					err := e.err
					vmErr := new(vm.VMError)
					if !errors.As(err, &vmErr) || vmErr.ErrorCode() != tt.wantVMErrCode {
						t.Errorf("err = %T(%v); want %T with ErrorCode() = %d", err, err, vmErr, tt.wantVMErrCode)
						if errors.Is(err, vmErr) {
							t.Logf("got %T.ErrorCode() = %d", vmErr, vmErr.ErrorCode())
						}
					}
				})
			}
		})
	}
}
