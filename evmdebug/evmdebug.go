// Package evmdebug provides debugging mechanisms for EVM contracts,
// intercepting opcode-level execution and allowing for inspection of data such
// as the VM's stack and memory.
package evmdebug

import (
	"context"

	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/solidifylabs/specops/internal/sync"
)

// NewDebugger constructs a new Debugger.
//
// Execution SHOULD be advanced until Debugger.Done() returns true otherwise
// resources will be leaked. Best practice is to always call FastForward(),
// usually in a deferred function.
//
// Debugger.State().Err SHOULD be checked once Debugger.Done() returns true.
//
// NOTE: see the limitations described in the Debugger comments.
func NewDebugger() *Debugger {
	step := make(chan step)
	fastForward := make(chan fastForward)
	stepped := make(chan stepped)
	done := make(chan done)

	// The outer and inner values have complementary send-receive abilities,
	// hence the duplication. This provides compile-time guarantees of intended
	// usage. The sending side is responsible for closing the channel.
	return &Debugger{
		step:        step,        // sent on to trigger a step
		fastForward: fastForward, // closed to trigger unblocked running
		stepped:     stepped,
		done:        done,
		d: &debugger{
			step:        step,
			fastForward: fastForward,
			stepped:     stepped, // sent on to signal end of single step
			done:        done,    // closed to signal end of running
		},
	}
}

// For stricter channel types as there are otherwise many with void types that
// can be accidentally switched.
type (
	step        struct{}
	fastForward struct{}
	stepped     struct{}
	done        struct{}
)

// A Debugger intercepts EVM opcode execution to allow inspection of the stack,
// memory, etc. The value returned by its Tracer() method should be placed
// inside a vm.Config before execution commences.
//
// Currently only a single frame is supported (i.e. no *CALL methods). This
// requires execution with a vm.EVMInterpreter.
type Debugger struct {
	d *debugger

	// Send external signals
	step        chan<- step
	fastForward chan<- fastForward
	// Receive internal state changes
	stepped <-chan stepped
	done    <-chan done
}

// Tracer returns an EVMLogger that enables debugging, compatible with geth.
//
// TODO: add an example demonstrating how to access vm.Config.
func (d *Debugger) Tracer() vm.EVMLogger {
	return d.d
}

// Wait blocks until Debugger is blocking the EVM from running the next opcode.
// The only reason to call Wait() is to access State() before the first Step().
func (d *Debugger) Wait() {
	d.waitForEVMBlocked()
}

func (d *Debugger) waitForEVMBlocked() {
	// Although use of context.Background() here goes against the style guide,
	// it is only because sync.Toggle requires a context. The blocking in this
	// case is guaranteed to be of negligible time so there's no point in
	// expecting users to pass a context to Wait().
	//
	// TODO: remove this once sync.Toggle has a non-context-aware option.

	// Deliberately dropping any error because only sync.ErrToggleClosed is
	// possible, which is a happy path for us.
	_ = d.d.blockingEVM.Wait(context.Background())
}

// close releases all resources; it MUST NOT be called before `done` is closed.
func (d *Debugger) close(closeFastForward bool) {
	close(d.step)
	if closeFastForward {
		close(d.fastForward)
	}
	d.d.blockingEVM.Close()
}

// Step advances the execution by one opcode. Step MUST NOT be called
// concurrently with any other Debugger methods. The first opcode is only
// executed upon the first call to Step(), allowing initial state to be
// inspected beforehand; see Wait() for this purpose.
//
// Step blocks until the opcode execution completes and the next opcode is being
// blocked.
//
// Step MUST NOT be called after Done() returns true.
func (d *Debugger) Step() {
	d.step <- step{}
	// CaptureState will either close d.done or toggle (off) and block d.Wait().
	// In both cases it performs the action *before* closing / sending on
	// this channel, so the checks in the select{} block are synchronised.
	<-d.stepped

	select {
	case <-d.done:
		d.close(true)
	default:
		// Fix for https://github.com/solidifylabs/specops/issues/25
		// When this unblocks we are guaranteed that the *next* opcode is being
		// blocked, which implies that the *current* one is finished, so we have
		// synchronised and can return.
		d.waitForEVMBlocked()
	}
}

// FastForward executes all remaining opcodes, effectively the same as calling
// Step() in a loop until Done() returns true.
//
// Unlike Step(), calling FastForward() when Done() returns true is acceptable.
// This allows it to be called in a deferred manner, which is best practice to
// avoid leaking resources:
//
//	dbg := evmdebug.NewDebugger()
//	defer dbg.FastForward()
func (d *Debugger) FastForward() {
	select {
	case <-d.d.fastForward: // already closed
		return
	default:
	}

	close(d.fastForward)
	for {
		select {
		case <-d.stepped: // gotta catch 'em all
		case <-d.done:
			d.close(false /*don't close d.fastForward again*/)
			return
		}
	}
}

// Done returns whether exeuction has ended.
func (d *Debugger) Done() bool {
	select {
	case <-d.done:
		return true
	default:
		return false
	}
}

// State returns the last-captured state, which will be modified upon each call
// to Step(). It is expected that State() only be called once, at any time after
// construction of the Debugger, and its result retained for inspection at each
// Step(). The CapturedState is, however, only valid after the first call to
// Step().
//
// Ownership of pointers is retained by the EVM instance that created
// them; modify with caution!
func (d *Debugger) State() *CapturedState {
	return &d.d.last
}

// CapturedState carries all values passed to the debugger.
//
// N.B. See ownership note in Debugger.State() documentation.
type CapturedState struct {
	PC, GasLeft, GasCost uint64
	Op                   vm.OpCode
	ScopeContext         *vm.ScopeContext // contains memory and stack ;)
	ReturnData           []byte
	Err                  error
}

// debugger implements vm.EVMLogger and is injected by its parent Debugger to
// intercept opcode execution.
type debugger struct {
	vm.EVMLogger // no need for most methods so just embed the interface

	// Waited upon by Capture{State,Fault}(), signalling an external call to
	// Step() or FastForward().
	step        <-chan step
	fastForward <-chan fastForward
	stepped     chan<- stepped
	// Toggled by Capture{State,Fault}(), externally signalling that the next
	// opcode is being blocked (also implying that the last one has completed,
	// allowing for synchronisation).
	blockingEVM sync.Toggle
	// Closed after execution of one of {STOP,RETURN,REVERT}, or upon a fault,
	// externally signalling completion of the execution.
	done chan<- done

	last CapturedState
}

// NOTE: when directly calling EVMInterpreter.Run(), only Capture{State,Fault}
// will ever be invoked.

func (d *debugger) CaptureState(pc uint64, op vm.OpCode, gasLeft, gasCost uint64, scope *vm.ScopeContext, retData []byte, depth int, err error) {
	d.blockingEVM.Set(true) // unblocks Debugger.Wait()

	// TODO: with the <-d.step at the beginning we can inspect initial state,
	// but what is actually available and how do we surface it?
	select {
	case <-d.step:
	case <-d.fastForward:
	}

	d.last.PC = pc
	d.last.Op = op
	d.last.GasLeft = gasLeft
	d.last.GasCost = gasCost
	d.last.ScopeContext = scope
	d.last.ReturnData = retData
	d.last.Err = err

	// In all cases below, closing / sending on d.stepped MUST be the last
	// action. Debugger.Step() relies on this to perform checks once its receive
	// on d.stepped is unblocked.
	switch op {
	case vm.STOP, vm.RETURN: // REVERT will end up in CaptureFault().
		close(d.done)
		close(d.stepped)
	default:
		d.blockingEVM.Set(false) // blocks Debugger.Wait()
		d.stepped <- stepped{}
	}
}

func (d *debugger) CaptureFault(pc uint64, op vm.OpCode, gasLeft, gasCost uint64, scope *vm.ScopeContext, depth int, err error) {
	d.blockingEVM.Set(true)
	defer func() { d.blockingEVM.Set(false) }()

	select {
	case <-d.step:
	case <-d.fastForward:
	}

	d.last.PC = pc
	d.last.Op = op
	d.last.GasLeft = gasLeft
	d.last.GasCost = gasCost
	d.last.ScopeContext = scope
	d.last.ReturnData = nil
	d.last.Err = err

	// See CaptureState for why closing d.stepped MUST be performed last.
	close(d.done)
	close(d.stepped)
}
