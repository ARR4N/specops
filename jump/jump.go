// Package jump provides jump-related opcodes (special and regular) for use in
// specops code.
package jump

import "fmt"

// A Dest is a Bytecoder that is converted into a vm.JUMPDEST while also storing
// its location in the bytecode for use via a PushDest or
// specops.PUSH[string|JUMPDEST|jump.Dest](<lbl>).
//
// Prefer specops.JUMPDEST, which is an alias of this type.
type Dest string

// Bytecode always returns an error as jump.Dest values have special handling
// inside Code.Compile().
func (d Dest) Bytecode() ([]byte, error) {
	return nil, fmt.Errorf("direct call to %T.Bytecode()", d)
}

// PushDest pushes the bytecode location of the respective Dest.
//
// Prefer specops.PUSHJUMPDEST, which is an alias of this type.
type PushDest string

// Bytecode always returns an error as PushDest values have special handling
// inside Code.Compile().
func (p PushDest) Bytecode() ([]byte, error) {
	return nil, fmt.Errorf("direct call to %T.Bytecode()", p)
}
