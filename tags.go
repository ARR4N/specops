package specops

import (
	"fmt"
	"unsafe"

	"github.com/arr4n/specops/types"
)

// A tag is a labelled location (byte index) in compiled code that can be
// referenced by the label instead of the numeric value. For example, a
// specops.JUMPDEST can be thought of as a tag followed by a
// vm.OpCode(JUMPDEST). Similarly, a Label is a lone tag.
type tag string

type tagged interface {
	types.Bytecoder
	tag() tag
}

var _ = []tagged{JUMPDEST(""), Label("")}

// A JUMPDEST is a Bytecoder that is converted into a vm.JUMPDEST while also
// storing its location in the bytecode for use via
// PUSH[string|JUMPDEST](<lbl>).
type JUMPDEST string

// Bytecode always returns an error as JUMPDEST values have special handling
// inside Code.Compile().
func (j JUMPDEST) Bytecode() ([]byte, error) {
	return nil, fmt.Errorf("direct call to %T.Bytecode()", j)
}

func (j JUMPDEST) tag() tag { return tag(j) }

// A Label marks a specific point in the code without adding any bytes when
// compiled. The corresponding numerical value is the first byte *after* the
// Label.
type Label string

// Bytecode always returns an error as Label values have special handling
// inside Code.Compile().
func (l Label) Bytecode() ([]byte, error) {
	return nil, fmt.Errorf("direct call to %T.Bytecode()", l)
}

func (l Label) tag() tag { return tag(l) }

// A pushTag pushes the tag's byte index to the stack.
type pushTag tag

func (p pushTag) Bytecode() ([]byte, error) {
	return nil, fmt.Errorf("direct call to %T.Bytecode()", p)
}

// A pushTags is the multi-tag equivalent of pushTag. It can be used, for
// example, for pushing jump tables.
type pushTags []tag

func (p pushTags) Bytecode() ([]byte, error) {
	return nil, fmt.Errorf("direct call to %T.Bytecode()", p)
}

func asPushTags[T ~string](xs []T) pushTags {
	return *(*pushTags)(unsafe.Pointer(&xs))
}

// Compile-time guarantee that tag itself has the same underlying type as those
// accepted by the generic function.
var _ = asPushTags[tag]

// PUSHSize pushes abs(loc(a),loc(b)), i.e. the size of the bytecode between the
// corresponding JUMPDEST(s) / Label(s).
func PUSHSize[T ~string, U ~string](a T, b U) types.Bytecoder {
	return pushSize{tag(a), tag(b)}
}

type pushSize [2]tag

func (p pushSize) Bytecode() ([]byte, error) {
	return nil, fmt.Errorf("direct call to %T.Bytecode()", p)
}
