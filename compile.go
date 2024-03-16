package specops

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/solidifylabs/specops/stack"
	"github.com/solidifylabs/specops/types"
)

// A splice is a (possibly empty) buffer of bytecode, followed by a JUMPDEST,
// PUSHJUMPDEST, or tablePusher. The location of a JUMPDEST changes the size of
// the other two, but the location isn't known until preceding push types are
// determined. A splice allows for lazy determination of locations.
type splice struct {
	buf bytes.Buffer
	op  types.Bytecoder // JUMPDEST, PUSHJUMPDEST, or tablePusher
	// If op is a JUMPDEST
	offset *int // Current estimate of offset in the bytecode, or nil if not yet estimated
	// If op is a PUSHJUMPDEST or tablePusher
	dests    []*splice // Splice(s) of the respective JUMPDEST(s)
	reserved int       // Number of bytes reserved (including the PUSH); 1 + (1 or 2) per dest
}

// bytesPerDest returns an optimistic estimate of the number of the least number
// of bytes needed to represent the largest s.dests.offset. If any non-nil
// offset >= 256 then bytesPerDest returns 2, otherwise it returns 1 (i.e. the
// optimistic element). This may change due to calls to spliceConcat.expand.
func (s *splice) bytesPerDest() int {
	for _, d := range s.dests {
		if d.offset != nil && *d.offset >= 256 {
			return 2
		}
	}
	return 1
}

// extraBytesNeeded returns the number of bytes needed to represent the
// JUMPDEST, PUSHJUMPDEST, or tablePusher of the splice.
func (s *splice) extraBytesNeeded() int {
	if s.op == nil { // final splice
		return 0
	}

	return 1 + len(s.dests)*s.bytesPerDest() - s.leadingZeroes()
}

// leadingZeroes returns the number of bytes that PUSH() will strip from the
// concatenated s.dests.offset values.
func (s *splice) leadingZeroes() int {
	if len(s.dests) == 0 {
		return 0
	}

	// In all cases, if d.offset is nil, it can never be set to 0 because that
	// would have had to already happened (by nature of being) the very first
	// opcode.

	if s.bytesPerDest() == 1 {
		var n int
		for _, d := range s.dests {
			if d.offset == nil || *d.offset != 0 {
				break
			}
			n++
		}
		return n
	}

	// bytesPerDest == 2
	var n int
	for _, d := range s.dests {
		switch {
		case d.offset == nil, *d.offset < 256:
			return n + 1
		case *d.offset == 0:
			n += 2
		default:
			return n
		}
	}
	return n
}

// A spliceConcat holds a set of sequential splices that are intended to be
// concatenated to produce bytecode. Its reserve() and expand() methods
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
// bytes.Buffer. This function MUST be called every time a JUMPDEST,
// PUSHJUMPDEST, or tablePusher is encountered by Code.Compile(), passing said
// op to be appended to the previous splice.
func newSpliceBuffer[T interface {
	JUMPDEST | PUSHJUMPDEST | tablePusher
}](s *spliceConcat, op T) *bytes.Buffer {
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
			toInvert := types.OpCode(op)
			// All DUP have the same upper nibble 0x8 and SWAP have 0x9.
			base := toInvert & 0xf0
			if base != vm.DUP1 && base != vm.SWAP1 {
				return nil, fmt.Errorf("%T applied to non-DUP/SWAP opcode %v", op, toInvert)
			}
			offset := toInvert - base

			last := types.OpCode(min(16, stackDepth))
			if base == SWAP1 {
				last--
			}
			if offset >= last {
				return nil, posErr("%T(%v) with stack depth %d", op, vm.OpCode(op), last)
			}

			use = base + last - offset - 1

			if b := use.(types.OpCode) & 0xf0; b != base {
				panic(fmt.Sprintf("BUG: bad inversion %v -> %v", vm.OpCode(op), vm.OpCode(use.(types.OpCode))))
			}

		case PUSHJUMPDEST:
			buf = newSpliceBuffer(splices, op)
			stackDepth++

		case tablePusher:
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
		case tablePusher:

		case Raw:
			code, _ := use.Bytecode() // always returns nil error
			buf.Write(code)

		default:
			code, err := use.Bytecode()
			if err != nil {
				return nil, err
			}

			for i, n := 0, len(code); i < n; i++ {
				op := vm.OpCode(code[i])
				d, ok := stackDeltas[op]
				if !ok {
					return nil, posErr("invalid %T(%v) as byte [%d] returned by Bytecode()", op, op, i)
				}
				if stackDepth < d.pop {
					return nil, posErr("Bytecode()[%d] popping %d values with stack depth %d", i, d.pop, stackDepth)
				}
				stackDepth += d.push - d.pop // we're not in Solidity anymore ;)

				if op.IsPush() {
					i += int(op - vm.PUSH0)
				}
			}

			buf.Write(code)
		}

	} // end CodeLoop

	if err := splices.reserve(); err != nil {
		return nil, err
	}
	if err := splices.expand(); err != nil {
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
// PUSH opcode. Similar logic applies to tablePushers as does to PUSHJUMPDESTs
// except that 2 bytes are reserved *per* JUMPDEST unless *all* have already
// been seen.
func (s *spliceConcat) reserve() error {
	var pc int
	for i, sp := range s.all {
		pc += sp.buf.Len()

		switch op := sp.op.(type) {
		case JUMPDEST:
			x := pc
			sp.offset = &x

		case PUSHJUMPDEST:
			d, ok := s.dests[JUMPDEST(op)]
			if !ok {
				return fmt.Errorf("%T(%q) without corresponding %T", op, op, JUMPDEST(""))
			}
			sp.dests = []*splice{d}

		case tablePusher:
			sp.dests = make([]*splice, len(op))
			for i, lbl := range op {
				d, ok := s.dests[lbl]
				if !ok {
					return fmt.Errorf("%T{…%q…} without corresponding %T", op, lbl, JUMPDEST(""))
				}
				sp.dests[i] = d
			}

		case nil:
			if i+1 != len(s.all) {
				return fmt.Errorf("BUG: %T with nil op MUST be last", sp)
			}
		}

		reserve := sp.extraBytesNeeded()
		pc += reserve
		sp.reserved = reserve
	}
	return nil
}

// expand performs one or more passes over all splices, finding PUSHJUMPDESTs
// and tablePushers with too few reserved bytes. This occurs when the respective
// JUMPDEST(s) were later in the code so their location(s) weren't yet known by
// reserve(). Every time the number of reserved bytes must be increased, an
// expansion counter is increased and later used on subsequent JUMPDESTs to move
// them back in the code.
//
// Note that PUSHJUMPDEST and tablePusher splices have pointers to their
// respective JUMPDEST splices so there is no need to adjust them to account for
// expansion. Only after expand() has returned will the pushed values be locked
// in.
//
// expand() MUST NOT be called before s.reserve().
//
// TODO: is there a more efficient algorithm? A cursory glance suggests that
// it's currently O(nm) for n PUSHs and m JUMPs, which is at least quadratic in
// n. The interplay between expansion via PUSHs and shifting of JUMPs suggests
// that this is best-possible, but perhaps early exiting is still possible.
func (s *spliceConcat) expand() error {
	for {
		expand := 0
		for _, sp := range s.all {
			switch sp.op.(type) {
			case JUMPDEST:
				*sp.offset += expand

			case nil:
				// last splice, as already checked in reserve()

			default:
				need := sp.extraBytesNeeded()
				if need > sp.reserved {
					expand += need - sp.reserved
					sp.reserved = need
				}
			}
		}

		if expand == 0 {
			return nil
		}
	}
}

// bytes returns the concatenated splices, with concrete PUSHJUMPDEST and
// tablePluser values. It SHOULD NOT be called before s.expand() and MUST NOT be
// called before s.reserve().
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

		case nil:
			// last splice

		default:
			// The leading zeroes will be stripped by PUSHBytes(), but we need
			// them to simplifying the binary-encoding loop.
			full := make([]byte, sp.extraBytesNeeded()+sp.leadingZeroes()-1) // -1 because the PUSH is separate
			buf := make([]byte, 8)

			n := sp.bytesPerDest()
			for i, dest := range sp.dests {
				binary.BigEndian.PutUint64(buf, uint64(*dest.offset))
				copy(full[i*n:(i+1)*n], buf[8-n:])
			}

			bc, err := PUSHBytes(full...).Bytecode()
			if err != nil {
				return nil, fmt.Errorf("pushing %T(%q): %v", sp.op, sp.op, err)
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
