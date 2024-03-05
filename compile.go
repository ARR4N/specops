package specops

import (
	"bytes"
	"fmt"

	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/solidifylabs/specops/stack"
	"github.com/solidifylabs/specops/types"
)

// A splice is a (possibly empty) buffer of bytecode, followed by either a
// JUMPDEST or PUSHJUMPDEST. The location of a JUMPDEST changes the size of a
// PUSHJUMPDEST, but the location isn't known until preceding PUSHJUMPDESTs are
// determine. A splice allows for lazy determination of locations.
type splice struct {
	buf bytes.Buffer
	op  types.Bytecoder // either JUMPDEST or PUSHJUMPDEST
	// If op is a JUMPDEST
	offset *int // Current estimate of offset in the bytecode, or nil if not yet estimated
	// If op is a PUSHJUMPDEST
	dest     *splice // Splice of the respective JUMPDEST
	reserved int     // Number of bytes, 2 or 3, reserved (including the PUSH)
}

// A spliceConcat holds a set of sequential splices that are intended to be
// concatenated to produce bytecode. Its reserve() and shrink() methods
// implement lazy instantiation of JUMPDEST locations.
type spliceConcat struct {
	all   []*splice
	dests map[JUMPDEST]*splice
}

// curr returns the last *splice in the spliceConcat.
func (s *spliceConcat) curr() *splice {
	if len(s.all) == 0 {
		return nil
	}
	return s.all[len(s.all)-1]
}

// newSpliceBuffer pushes a new *splice to the spliceConcat and returns its
// bytes.Buffer. This function MUST be called every time a JUMPDEST or
// PUSHJUMPDEST is encountered by Code.Compile(), passing said op to be appended
// to the previous splice.
func newSpliceBuffer[T interface{ JUMPDEST | PUSHJUMPDEST }](s *spliceConcat, op T) *bytes.Buffer {
	curr := s.curr()
	curr.op = types.Bytecoder(op)
	if j, ok := any(op).(JUMPDEST); ok {
		s.dests[j] = curr
	}

	s.all = append(s.all, new(splice))
	return &s.curr().buf
}

// flatten returns a Code slice that only contains Bytecoders but no
// BytecodeHolders, the latter being recursively converted into their
// constituent Bytecoders.
func (c Code) flatten() Code {
	out := make(Code, 0, len(c))
	for _, bc := range c {
		switch bc := bc.(type) {
		case types.BytecodeHolder:
			out = append(out, Code(bc.Bytecoders()).flatten()...)
		default:
			out = append(out, bc)
		}
	}
	return out
}

// Compile returns a compiled EVM contract with all special opcodes interpreted.
func (c Code) Compile() ([]byte, error) {
	flat := c.flatten()

	splices := &spliceConcat{
		all:   []*splice{new(splice)},
		dests: make(map[JUMPDEST]*splice),
	}
	buf := &splices.all[0].buf

	var (
		stackDepth               uint
		requireStackDepthSetting bool
	)

CodeLoop:
	for i, raw := range flat {
		use := raw

		posErr := func(format string, a ...any) error {
			format = "%T[%d]: " + format
			a = append([]any{c, i}, a...)
			return fmt.Errorf(format, a...)
		}

		switch op := raw.(type) {
		case stack.SetDepth:
			stackDepth = uint(op)
			requireStackDepthSetting = false
			continue CodeLoop

		case stack.ExpectDepth:
			if got, want := stackDepth, uint(op); got != want {
				return nil, posErr("stack depth %d when expecting %d", got, want)
			}
			continue CodeLoop

		case Inverted:
			toInvert := opCode(op)
			// All DUP have the same upper nibble 0x8 and SWAP have 0x9.
			base := toInvert & 0xf0
			if base != vm.DUP1 && base != vm.SWAP1 {
				return nil, fmt.Errorf("%T applied to non-DUP/SWAP opcode %v", op, toInvert)
			}
			offset := toInvert - base

			last := opCode(min(16, stackDepth))
			if base == SWAP1 {
				last--
			}
			if offset >= last {
				return nil, posErr("%T(%v) with stack depth %d", op, vm.OpCode(op), last)
			}

			use = base + last - offset - 1

			if b := use.(opCode) & 0xf0; b != base {
				panic(fmt.Sprintf("BUG: bad inversion %v -> %v", vm.OpCode(op), vm.OpCode(use.(opCode))))
			}

		case PUSHJUMPDEST:
			buf = newSpliceBuffer(splices, op)
			stackDepth++

		case JUMPDEST:
			if _, ok := splices.dests[op]; ok {
				return nil, fmt.Errorf("duplicate JUMPDEST label %q", op)
			}
			buf = newSpliceBuffer(splices, op)

		} // end switch raw.(type)

		if requireStackDepthSetting {
			return nil, posErr("%T must be followed by %T", JUMPDEST(""), stack.SetDepth(0))
		}

		switch raw.(type) {
		case JUMPDEST:
			requireStackDepthSetting = true

		case PUSHJUMPDEST:

		case Raw:
			code, _ := use.Bytecode() // always returns nil error
			buf.Write(code)

		default:
			code, err := use.Bytecode()
			if err != nil {
				return nil, err
			}

			op := vm.OpCode(code[0])
			d, ok := stackDeltas[op]
			if !ok {
				return nil, posErr("invalid %T(%v) as first byte returned by Bytecode()", op, op)
			}
			if stackDepth < d.pop {
				return nil, posErr("popping %d values with stack depth %d", d.pop, stackDepth)
			}
			stackDepth += d.push - d.pop // we're not in Solidity anymore ;)

			buf.Write(code)
		}

	} // end CodeLoop

	if err := splices.reserve(); err != nil {
		return nil, err
	}
	if err := splices.shrink(); err != nil {
		return nil, err
	}
	return splices.bytes()
}

// reserve performs a single pass over all splices, recording a worst-case
// offset for each JUMPDEST. If a PUSHJUMPDEST refers to an already-seen
// JUMPDEST, either 1 or 2 bytes are reserved, based on said JUMPDEST's
// recorded offset. If a PUSHJUMPDEST refers to a not-yet-seen JUMPDEST, 2 bytes
// are reserved as the JUMPDEST's offset may be beyond the 256th byte. Two bytes
// are sufficient to record any position in 64KiB (larger than all possible
// contracts). An extra byte is reserved for all PUSHJUMPDESTs for the actual
// PUSH opcode.
func (s *spliceConcat) reserve() error {
	var pc int
	for i, sp := range s.all {
		pc += sp.buf.Len()

		switch op := sp.op.(type) {
		case JUMPDEST:
			x := pc
			sp.offset = &x
			pc++

		case PUSHJUMPDEST:
			d, ok := s.dests[JUMPDEST(op)]
			if !ok {
				return fmt.Errorf("%T(%q) without corresponding %T", op, op, JUMPDEST(""))
			}

			reserve := 3 // PUSHn and 2 bytes as worst-case estimate
			if d.offset != nil && *d.offset < 256 {
				reserve = 2
			}
			pc += reserve
			sp.reserved = reserve
			sp.dest = d

		case nil:
			if i != len(s.all)-1 {
				return fmt.Errorf("BUG: %T with nil op MUST be last", sp)
			}
		}
	}
	return nil
}

// shrink performs one or more passes over all splices, finding PUSHJUMPDESTs
// with too many reserved bytes. This occurs when the respective JUMPDEST was
// later in the code so its location wasn't yet known by reserve(). Every time
// the number of reserved bytes can be reduced, a shrinkage counter is
// incremented and later used on subsequent JUMPDESTs to move them forward in
// the code.
//
// Note that PUSHJUMPDEST splices have pointers to their respective JUMPDEST
// splices so there is no need to adjust them to account for shrinkage. Only
// after shrink() has returned will the pushed values be locked in.
//
// shrink MUST NOT be called before s.reserve().
//
// TODO: is there a more efficient algorithm? A cursory glance suggests that
// it's currently O(nm) for n PUSHs and m JUMPs, which is at least quadratic in
// n. The interplay between shrinkage via PUSHs and shifting of JUMPs suggests
// that this is best-possible, but perhaps early exiting is still possible.
func (s *spliceConcat) shrink() error {
	for {
		var shrink int
		for _, sp := range s.all {
			switch sp.op.(type) {
			case JUMPDEST:
				*sp.offset -= shrink

			case PUSHJUMPDEST:
				dest := sp.dest

				need := 3
				if *dest.offset < 256 {
					need = 2
				}
				if need < sp.reserved {
					sp.reserved--
					shrink++
				}
			}
		}

		if shrink == 0 {
			return nil
		}
	}
}

// bytes returns the concatenated splices, with concrete PUSHJUMPDEST values. It
// SHOULD NOT be called before s.shrink() and MUST NOT be called before
// s.reserve().
func (s *spliceConcat) bytes() ([]byte, error) {
	code := new(bytes.Buffer)
	for _, sp := range s.all {
		if _, err := sp.buf.WriteTo(code); err != nil {
			// This should be impossible, but ignoring the error angers the
			// linter.
			return nil, fmt.Errorf("%T.bytes(): %T.buf.WriteTo(%T): %v", s, sp, code, err)
		}

		switch sp.op.(type) {
		case JUMPDEST:
			code.WriteByte(byte(vm.JUMPDEST))

		case PUSHJUMPDEST:
			bc, err := PUSH(uint64(*sp.dest.offset)).Bytecode()
			if err != nil {
				return nil, fmt.Errorf("BUG: pushing JUMPDEST %q: %v", sp.op, err)
			}
			code.Write(bc)
		}
	}
	return code.Bytes(), nil
}

func min(a, b uint) uint {
	if a <= b {
		return a
	}
	return b
}
