// Package specops implements a DSL for crafting raw EVM bytecode. It
// provides "special" opcodes as drop-in replacements for regular ones, e.g.
// JUMPDEST labels, PUSH<N> aliases, and DUP/SWAP from the bottom of the stack.
// It also provides pseudo opcodes that act as compiler hints.
//
// It is designed to be dot-imported such that all exported identifiers are
// available in the importing package, allowing a mnemonic-style programming
// environment akin to writing assembly. As a result, there are few top-level
// identifiers.
package specops

import (
	"encoding/binary"
	"fmt"
	"math/bits"
	"unsafe"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"

	"github.com/solidifylabs/specops/types"
)

// Code is a slice of Bytecoders; it is itself a Bytecoder, allowing for
// nesting.
type Code []types.Bytecoder

// Bytecode always returns an error; use Code.Compile instead(), which flattens
// nested Code instances.
func (c Code) Bytecode() ([]byte, error) {
	return nil, fmt.Errorf("call to %T.Bytecode()", c)
}

// Bytecoders returns the Code as a slice of Bytecoders.
func (c Code) Bytecoders() []types.Bytecoder {
	return []types.Bytecoder(c)
}

// Fn returns a Bytecoder that returns the concatenation of the *reverse* of
// bcs. This allows for a more human-readable syntax akin to a function call
// (hence the name). Fn is similar to Yul except that "return" values are left
// on the stack to be used by later Fn()s (or raw bytecode).
//
// Although the returned BytecodeHolder can contain JUMPDESTs, they're hard to
// reason about so should be used with care.
func Fn(bcs ...types.Bytecoder) types.BytecodeHolder {
	c := Code(bcs)
	for i, n := 0, len(c); i < n/2; i++ {
		j := n - i - 1
		c[i], c[j] = c[j], c[i]
	}
	return c
}

// Raw is a Bytecoder that bypasses all compiler checks and simply appends its
// contents to bytecode. It can be used for raw data, not meant to be executed.
type Raw []byte

// Bytecode returns `r` unchanged, and a nil error.
func (r Raw) Bytecode() ([]byte, error) {
	return []byte(r), nil
}

// A label is a type that tags a section of code with a (human-readable) string
// that can be PUSH()d.
type label interface {
	types.Bytecoder
	label() string
}

var _ = []label{JUMPDEST(""), Label("")}

// A JUMPDEST is a Bytecoder that is converted into a vm.JUMPDEST while also
// storing its location in the bytecode for use via
// PUSH[string|JUMPDEST](<lbl>).
type JUMPDEST string

// Bytecode always returns an error as JUMPDEST values have special handling
// inside Code.Compile().
func (j JUMPDEST) Bytecode() ([]byte, error) {
	return nil, fmt.Errorf("direct call to %T.Bytecode()", j)
}

func (j JUMPDEST) label() string {
	return string(j)
}

// A Label marks a specific point in the code without adding any bytes when
// compiled. The corresponding numerical value is the first byte *after* the
// Label.
type Label string

// Bytecode always returns an error as Label values have special handling
// inside Code.Compile().
func (l Label) Bytecode() ([]byte, error) {
	return nil, fmt.Errorf("direct call to %T.Bytecode()", l)
}

func (l Label) label() string {
	return string(l)
}

// A pushLabel pushes the respective JUMPDEST/Label to the stack.
type pushLabel string

func (p pushLabel) Bytecode() ([]byte, error) {
	return nil, fmt.Errorf("direct call to %T.Bytecode()", p)
}

// A pushLabels is the multi-label equivalent of pushLabel. It can be used, for
// example, for pushing jump tables.
type pushLabels []string

func (p pushLabels) Bytecode() ([]byte, error) {
	return nil, fmt.Errorf("direct call to %T.Bytecode()", p)
}

func asPushLabels[T ~string](xs []T) pushLabels {
	return *(*pushLabels)(unsafe.Pointer(&xs))
}

// PUSHSelector returns a PUSH4 Bytecoder that pushes the selector of the
// signature, i.e. `sha3(sig)[:4]`.
func PUSHSelector(sig string) types.Bytecoder {
	return PUSH(crypto.Keccak256([]byte(sig))[:4])
}

// PUSHBytes accepts [1,32] bytes, returning a PUSH<x> Bytecoder where x is the
// smallest number of bytes (possibly zero) that can represent the concatenated
// values; i.e. x = len(bs) - leadingZeros(bs).
func PUSHBytes(bs ...byte) types.Bytecoder {
	return types.BytecoderFromStackPusher(bytesPusher(bs))
}

type bytesPusher []byte

func (p bytesPusher) ToPush() []byte { return []byte(p) }

// PUSH returns a PUSH<n> Bytecoder appropriate for the type. It panics if v is
// negative. A string refers to the respective JUMPDEST or Label while a
// []string refers to a concatenation of the same (e.g. a JUMP table).
func PUSH[P interface {
	int | uint64 | common.Address | uint256.Int | byte | []byte | JUMPDEST | []JUMPDEST | Label | []Label | string | []string
}](v P,
) types.Bytecoder {
	pToB := types.BytecoderFromStackPusher

	switch v := any(v).(type) {
	case int:
		if v < 0 {
			panic(fmt.Sprintf("PUSH() negative value %d", v))
		}
		return pToB(uint64Pusher(uint64(v)))

	case uint64:
		return pToB(uint64Pusher(v))

	case byte:
		return PUSHBytes(v)

	case []byte:
		return PUSHBytes(v...)

	case common.Address:
		return pToB(addressPusher(v))

	case uint256.Int:
		return pToB(wordPusher(v))

	case string:
		return pushLabel(v)

	case JUMPDEST:
		return pushLabel(v)

	case []JUMPDEST:
		return asPushLabels(v)

	case Label:
		return pushLabel(v)

	case []Label:
		return asPushLabels(v)

	case []string:
		return pushLabels(v)

	default:
		panic(fmt.Sprintf("no type-switch for %T", v))
	}
}

// A uint64Pusher converts itself into the smallest possible representation in
// bytes.
type uint64Pusher uint64

func (p uint64Pusher) ToPush() []byte {
	if p == 0 {
		return []byte{0}
	}
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(p))
	return b[bits.LeadingZeros64(uint64(p))/8:]
}

type wordPusher uint256.Int

func (p wordPusher) ToPush() []byte {
	i := (*uint256.Int)(&p)
	if i.IsZero() {
		return []byte{0}
	}
	return i.Bytes()
}

type addressPusher common.Address

func (p addressPusher) ToPush() []byte { return p[:] }
