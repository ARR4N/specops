package runopts

import (
	"sync"

	"github.com/ethereum/go-ethereum/core/vm"
)

// NewDebugger constructs a new Debugger.
//
// Execution SHOULD be advanced until Debugger.Done() returns true otherwise
// resources will be leaked. Best practice is to always call FastForward(),
// usually in a deferred function.
//
// Debugger.State().Err SHOULD be checked once Debugger.Done() returns true.
func NewDebugger() *Debugger {
	started := make(chan started)
	step := make(chan step)
	fastForward := make(chan fastForward)
	stepped := make(chan stepped)
	done := make(chan done)

	// The outer and inner values have complementary send-receive abilities,
	// hence the duplication. This provides compile-time guarantees of intended
	// usage. The sending side is responsible for closing the channel.
	return &Debugger{
		started:     started,
		step:        step,
		fastForward: fastForward,
		stepped:     stepped,
		done:        done,
		d: &debugger{
			started:     started,
			step:        step,
			fastForward: fastForward,
			stepped:     stepped,
			done:        done,
		},
	}
}

// For stricter channel types as there are otherwise many with void types that
// can be accidentally switched.
type (
	started     struct{}
	step        struct{}
	fastForward struct{}
	stepped     struct{}
	done        struct{}
)

// A Debugger is an Option that intercepts opcode execution to allow inspection
// of the stack, memory, etc.
type Debugger struct {
	d *debugger

	// Send external signals
	step        chan<- step
	fastForward chan<- fastForward
	// Receive internal state changes
	started <-chan started
	stepped <-chan stepped
	done    <-chan done
}

// Apply adds a VMConfig.Tracer to the Configuration, intercepting execution of
// every opcode.
func (d *Debugger) Apply(c *Configuration) error {
	c.VMConfig.Tracer = d.d
	return nil
}

// Wait blocks until the bytecode is ready for execution, but the first opcode
// is yet to be executed; see Step().
func (d *Debugger) Wait() {
	<-d.started
}

// close releases all resources; it MUST NOT be called before `done` is closed.
func (d *Debugger) close(closeFastForward bool) {
	close(d.step)
	if closeFastForward {
		close(d.fastForward)
	}
}

// Step advances the execution by one opcode. Step MUST NOT be called
// concurrently with any other Debugger methods. The first opcode is only
// executed upon the first call to Step(), allowing initial state to be
// inspected beforehand.
//
// Step MUST NOT be called after Done() returns true.
func (d *Debugger) Step() {
	d.step <- step{}
	<-d.stepped

	select {
	case <-d.done:
		d.close(true)
	default:
	}
}

// FastForward executes all remaining opcodes, effectively the same as calling
// Step() in a loop until Done() returns true.
//
// Unlike Step(), calling FastForward() when Done() returns true is acceptable.
// This allows it to be called in a deferred manner, which is best practice to
// avoid leaking resources.
func (d *Debugger) FastForward() {
	select {
	case <-d.d.fastForward: // already closed:
		return
	default:
	}

	close(d.fastForward)
	for {
		select {
		case <-d.stepped: // gotta catch 'em all
		case <-d.done:
			d.close(false)
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

	// Waited upon by CaptureState(), signalling an external call to Step().
	step        <-chan step
	fastForward <-chan fastForward
	stepped     chan<- stepped
	// Closed by Capture{State,Fault}(), externally signalling the start of
	// execution.
	started   chan<- started
	startOnce sync.Once
	// Closed after execution of one of {STOP,RETURN,REVERT}, or upon a fault,
	// externally signalling completion of the execution.
	done chan<- done

	last CapturedState
}

func (d *debugger) setStarted() {
	d.startOnce.Do(func() {
		close(d.started)
	})
}

// NOTE: when directly calling EVMInterpreter.Run(), only Capture{State,Fault}
// will ever be invoked.

func (d *debugger) CaptureState(pc uint64, op vm.OpCode, gasLeft, gasCost uint64, scope *vm.ScopeContext, retData []byte, depth int, err error) {
	d.setStarted()

	// TODO: with the <-d.step at the beginning we can inspect initial state,
	// but what is actually available and how do we surface it? Perhaps Apply()
	// can keep a copy of the *Configuration and access the StateDB.
	select {
	case <-d.step:
	case <-d.fastForward:
	}

	defer func() {
		switch op {
		case vm.STOP, vm.RETURN: // REVERT will end up in CaptureFault().
			// Unlike d.started, we don't use a sync.Once for this because
			// if it's called twice then we have a bug and want to know
			// about it.
			close(d.stepped)
			close(d.done)
		default:
			d.stepped <- stepped{}
		}
	}()

	d.last.PC = pc
	d.last.Op = op
	d.last.GasLeft = gasLeft
	d.last.GasCost = gasCost
	d.last.ScopeContext = scope
	d.last.ReturnData = retData
	d.last.Err = err
}

func (d *debugger) CaptureFault(pc uint64, op vm.OpCode, gasLeft, gasCost uint64, scope *vm.ScopeContext, depth int, err error) {
	d.setStarted()
	defer func() { close(d.done) }()

	d.last.PC = pc
	d.last.Op = op
	d.last.GasLeft = gasLeft
	d.last.GasCost = gasCost
	d.last.ScopeContext = scope
	d.last.ReturnData = nil
	d.last.Err = err
}
