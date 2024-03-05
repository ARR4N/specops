package stack

import "fmt"

// ExpectDepth is a sentinel value that singals to Code.Compile() that it must
// assert the expected stack depth, returning an error if incorrect. See
// SetDepth() for caveats; note that the expectation is with respect to
// specops.Code.Compile() and has nothing to do with concrete (runtime) depths.
type ExpectDepth uint

// Bytecode always returns an error.
func (d ExpectDepth) Bytecode() ([]byte, error) {
	return nil, fmt.Errorf("call to %T.Bytecode()", d)
}

// SetDepth is a sentinel value that signals to specops.Code.Compile() that it
// must modify its internal counter reflecting the current stack depth.
//
// For each vm.OpCode that it encounters, Code.Compile() adjusts a value that
// reflects its belief about the stack depth. This is a crude mechanism that
// only works for non-JUMPing code. The programmer can therefore signal,
// typically after a JUMPDEST, the actual stack depth.
type SetDepth uint

// Bytecode always returns an error.
func (d SetDepth) Bytecode() ([]byte, error) {
	return nil, fmt.Errorf("call to %T.Bytecode()", d)
}
