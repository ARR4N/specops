package specops

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"

	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/arr4n/specops/stack"
	"github.com/arr4n/specops/types"
)

type lazyLocator interface {
	types.Bytecoder
	lazy()
}

func (JUMPDEST) lazy() {}
func (Label) lazy()    {}
func (pushTag) lazy()  {}
func (pushTags) lazy() {}
func (pushSize) lazy() {}

// A splice is a (possibly empty) buffer of bytecode, followed by a lazyLocator.
// The location of a tag changes the size of pushTags{s} that refer to it, but
// the location isn't known until preceding pushTag{s} are determined. A splice
// allows for lazy determination of locations.
type splice struct {
	buf bytes.Buffer
	op  lazyLocator
	// If tag
	offset *int // Current estimate of offset in the bytecode, or nil if not yet estimated
	// If pushTag{s}
	tags     []*splice // All have `op` field of type `tagged`
	reserved int       // Number of bytes reserved (including the PUSH); 1 + (1 or 2) per tag
}

// setTags populates splice.tags with the each of the `tags`, sourced from the
// `known` set.
func (s *splice) setTags(known map[tag]*splice, tags ...tag) error {
	var wantN int
	switch s.op.(type) {
	case pushTag: // singular
		wantN = 1
	case pushTags: // plural
		wantN = len(tags)
	case pushSize:
		wantN = 2
	default:
		return fmt.Errorf("BUG: %T.setTags() with unsupported %T op", s, s.op)
	}
	if n := len(tags); n != wantN {
		return fmt.Errorf("BUG: %T.setTags() with %T op; got %d tags; MUST be %d", s, s.op, n, wantN)
	}

	s.tags = make([]*splice, len(tags))
	for i, t := range tags {
		k, ok := known[t]
		if !ok {
			return fmt.Errorf("%T{…%q…} without corresponding %T/%T", s.op, tags, JUMPDEST(""), Label(""))
		}
		s.tags[i] = k
	}
	return nil
}

// bytesPerTag returns an optimistic estimate of the least number of bytes
// needed to represent the largest s.tags.offset. If any non-nil offset is >=256
// then bytesPerTag returns 2, otherwise it returns 1 (i.e. the optimistic
// element). This may change due to calls to spliceConcat.expand.
func (s *splice) bytesPerTag() int {
	if _, ok := s.op.(pushSize); ok {
		// If this happens then there's a broken invariant; this is never
		// expected to happen in production code so a panic is ok per:
		// https://google.github.io/styleguide/go/best-practices#when-to-panic
		panic(fmt.Sprintf("BUG: %T.bytesPerTag() with %T; use bytesForSize()", s, pushSize{}))
	}

	for _, t := range s.tags {
		if t.offset != nil && *t.offset >= 256 {
			return 2
		}
	}
	return 1
}

// bytesForSize is the equivalent of bytesPerTag(), assuming s.op is a pushSize.
// Instead of performing calculations based on a tag offset, it uses the
// absolute difference between the two.
func (s *splice) bytesForSize() int {
	if _, ok := s.op.(pushSize); !ok {
		// See rationale in bytesPerTag().
		panic(fmt.Sprintf("BUG: %T.bytesPerSize() with %T; use bytesForTag()", s, pushSize{}))
	} else if n := len(s.tags); n != 2 {
		panic(fmt.Sprintf("BUG: %T.bytesPerSize() with %d tags; MUST be 2", s, n))
	}

	t0, t1 := s.tags[0].offset, s.tags[1].offset
	if t0 != nil && t1 != nil && absDiff(*t0, *t1) >= 256 {
		return 2
	}
	return 1
}

// extraBytesNeeded returns the number of bytes needed to represent the
// lazyLocator of the splice, over and above the splice's buffer. This includes
// the single byte for the actual PUSHn opcode.
func (s *splice) extraBytesNeeded() int {
	switch s.op.(type) {
	case nil: // final splice
		return 0

	case Label:
		return 0

	case pushSize:
		return 1 + s.bytesForSize() - s.leadingZeroes()

	default:
		return 1 + len(s.tags)*s.bytesPerTag() - s.leadingZeroes()
	}

}

// leadingZeroes returns the number of bytes that PUSH() will strip from the
// concatenated s.tags.offset values of pushTag{s}.
func (s *splice) leadingZeroes() int {
	if len(s.tags) == 0 {
		return 0
	}

	if _, ok := s.op.(pushSize); ok {
		n := s.bytesForSize()
		t0 := s.tags[0].offset
		t1 := s.tags[1].offset

		switch {
		case t0 == nil || t1 == nil || *t0 == *t1:
			return n

		case n == 1:
			return 0

		case n == 2:
			if absDiff(*t0, *t1) < 256 {
				return 1
			}
			return 0
		}

		// See rationale for panic in bytesForTag().
		panic(fmt.Sprintf("BUG: unsupported branch in %T.leadingZeroes() for %T op", s, s.op))
	}

	// In all cases, if t.offset is nil, it can never be set to 0 because that
	// would have had to already happened (by nature of being) the very first
	// opcode.

	if s.bytesPerTag() == 1 {
		var n int
		for _, t := range s.tags {
			if t.offset == nil || *t.offset != 0 {
				break
			}
			n++
		}
		return n
	}

	// bytesPerTag == 2
	var n int
	for _, t := range s.tags {
		switch {
		case t.offset == nil, *t.offset < 256:
			return n + 1
		case *t.offset == 0:
			n += 2
		default:
			return n
		}
	}
	return n
}

// A spliceConcat holds a set of sequential splices that are intended to be
// concatenated to produce bytecode. Its reserve() and expand() methods
// implement lazy instantiation of tag locations.
type spliceConcat struct {
	splices []*splice
	allTags map[tag]*splice
}

// curr returns the last *splice in the spliceConcat.
func (s *spliceConcat) curr() *splice {
	if len(s.splices) == 0 {
		return nil
	}
	return s.splices[len(s.splices)-1]
}

// newSpliceBuffer pushes a new *splice to the spliceConcat and returns its
// bytes.Buffer. This function MUST be called every time a lazyLocator is
// encountered by Code.Compile(), passing said op to be appended to the previous
// splice.
func newSpliceBuffer(s *spliceConcat, op lazyLocator) (*bytes.Buffer, error) {
	curr := s.curr()
	curr.op = op

	if l, ok := op.(tagged); ok {
		if _, ok := s.allTags[l.tag()]; ok {
			return nil, fmt.Errorf("duplicate JUMPDEST/Label %q", op)
		}
		s.allTags[l.tag()] = curr
	}

	s.splices = append(s.splices, new(splice))
	return &s.curr().buf, nil
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
		splices: []*splice{new(splice)},
		allTags: make(map[tag]*splice),
	}
	buf := &splices.splices[0].buf

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

		case lazyLocator:
			b, err := newSpliceBuffer(splices, op)
			if err != nil {
				return nil, err
			}
			buf = b

			if _, ok := op.(tagged); !ok {
				// Not a tag itself therefore must be pushing one to the stack.
				stackDepth++
			}

		} // end switch raw.(type)

		if requireStackDepthSetting {
			return nil, posErr("%T must be followed by %T", JUMPDEST(""), stack.SetDepth(0))
		}

		switch raw.(type) {
		case JUMPDEST:
			requireStackDepthSetting = true

		case lazyLocator:

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

// reserve performs a single pass over all splices, recording a best-case
// offset for each tagged location. If a pushTag refers to an already-seen
// tag, either 1 or 2 bytes are reserved, based on said tag's recorded offset.
// If a pushTag refers to an unseen tag, 1 byte is reserved as an optimistic
// estimate of the offset, which may be increased by expand(). An extra byte is
// reserved for each `pushTag` for the actual PUSH opcode. Similar logic applies
// to `pushTags` as does to individual `pushTag`s except that 1 byte is reserved
// *per* tag unless *all* have already been seen to be at or beyond the 256th
// byte.
func (s *spliceConcat) reserve() error {
	var pc int
	for i, sp := range s.splices {
		pc += sp.buf.Len()

		switch op := sp.op.(type) {
		case tagged: // JUMPDEST or Label
			x := pc
			sp.offset = &x

		case pushTag:
			if err := sp.setTags(s.allTags, tag(op)); err != nil {
				return err
			}

		case pushTags:
			if err := sp.setTags(s.allTags, op...); err != nil {
				return err
			}

		case pushSize:
			if err := sp.setTags(s.allTags, op[0], op[1]); err != nil {
				return err
			}

		case nil:
			if i+1 != len(s.splices) {
				return fmt.Errorf("BUG: %T with nil op MUST be last", sp)
			}

		default:
			return fmt.Errorf("BUG: %T.reserve() encountered %T.op of unsupported type %T", s, sp, op)
		}

		reserve := sp.extraBytesNeeded()
		pc += reserve
		sp.reserved = reserve
	}
	return nil
}

// expand performs one or more passes over all splices, finding `pushTag`s and
// `pushTags` with too few reserved bytes. This occurs when the respective
// tagged locations were later in the code so their offset(s) weren't yet known
// by reserve(). Every time the number of reserved bytes must be increased, an
// expansion counter is increased and later used on subsequent tags to move them
// later in the code.
//
// Note that pushTag{s} splices have pointers to the splices of their respective
// tags so there is no need to adjust them to account for expansion. Only after
// expand() has returned will the pushed values be locked in.
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
		for _, sp := range s.splices {
			switch sp.op.(type) {
			case tagged:
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

// bytes returns the concatenated splices, with concrete pushTag{s} values. It
// MUST NOT be called before s.reserve() nor s.expand().
func (s *spliceConcat) bytes() ([]byte, error) {
	code := new(bytes.Buffer)
	for _, sp := range s.splices {
		if _, err := sp.buf.WriteTo(code); err != nil {
			// This should be impossible, but ignoring the error angers the
			// linter.
			return nil, fmt.Errorf("%T.bytes(): %T.buf.WriteTo(%T): %v", s, sp, code, err)
		}

		switch op := sp.op.(type) {
		case JUMPDEST:
			code.WriteByte(byte(vm.JUMPDEST))

		case Label: // purely for labelling, not adding to the code
		case nil: // last splice

		case pushSize:
			diff := uint64(absDiff(*sp.tags[0].offset, *sp.tags[1].offset))
			if diff > math.MaxUint16 {
				// A contract this large couldn't be deployed (at least on Mainnet)
				return nil, fmt.Errorf("pushing size between %q and %q; %d can't be represented with 2 bytes", op[0], op[1], diff)
			}

			bc, err := PUSHBytes(byte(diff>>8), byte(diff)).Bytecode()
			if err != nil {
				return nil, fmt.Errorf("pushing size %d between %q and %q: %v", diff, op[0], op[1], err)
			}
			code.Write(bc)

		default:
			// The leading zeroes will be stripped by PUSHBytes(), but we need
			// them to simplify the binary-encoding loop.
			full := make([]byte, sp.extraBytesNeeded()+sp.leadingZeroes()-1) // -1 because the PUSH is separate
			buf := make([]byte, 8)

			n := sp.bytesPerTag()
			for i, tag := range sp.tags {
				binary.BigEndian.PutUint64(buf, uint64(*tag.offset))
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

func absDiff(i, j int) int {
	switch d := i - j; {
	case d < 0:
		return -d
	default:
		return d
	}
}

func min(a, b uint) uint {
	if a <= b {
		return a
	}
	return b
}
