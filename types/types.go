// Package types defines types used by the specialops package, which is intended
// to be dot-imported so requires a minimal footprint of exported symbols.
package types

import (
	"fmt"

	"github.com/ethereum/go-ethereum/core/vm"
)

// A Bytecoder returns raw EVM bytecode. If the returned bytecode is the
// concatenation of multiple Bytecoder outputs, the type MUST also implement
// BytecodeHolder.
type Bytecoder interface {
	Bytecode() ([]byte, error)
}

// A BytecodeHolder is a concatenation of Bytecoders.
type BytecodeHolder interface {
	Bytecoder
	Bytecoders() []Bytecoder
}

// A StackPusher returns [1,32] bytes to be pushed to the stack.
type StackPusher interface {
	ToPush() []byte
}

// BytecoderFromStackPusher returns a Bytecoder that calls s.ToPush() and
// prepends the appropriate PUSH<N> opcode to the returned bytecode.
func BytecoderFromStackPusher(s StackPusher) Bytecoder {
	return pusher{s}
}

type pusher struct {
	StackPusher
}

func (p pusher) Bytecode() ([]byte, error) {
	buf := p.ToPush()
	n := len(buf)
	if n == 0 || n > 32 {
		return nil, fmt.Errorf("len(%T.ToPush()) == %d must be in [1,32]", p.StackPusher, n)
	}

	size := n
	for _, b := range buf {
		if b == 0 {
			size--
		} else {
			break
		}
	}
	if size == 0 {
		return []byte{byte(vm.PUSH0)}, nil
	}

	return append(
		// PUSH0 to PUSH32 are contiguous, so we can perform arithmetic on them.
		[]byte{byte(vm.PUSH0 + vm.OpCode(size))},
		buf[n-size:]...,
	), nil
}
