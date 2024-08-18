package stack

import (
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/arr4n/specops/types"
)

type xFormType int

const (
	unknownXform xFormType = iota
	permutation
	general
)

// A Transformation transforms the stack by modifying its order, growing, and/or
// shrinking it.
type Transformation struct {
	typ      xFormType
	depth    uint8
	indices  []uint8
	override []types.OpCode
}

// Permute returns a Transformation that permutes the order of the stack. The
// indices MUST be a contiguous set of distinct values [0,n) in any order.
//
// While permutations can also be achieved with Transform(), Permute performs
// additional checks on the indices (guaranteeing only SWAPs) and signals intent
// to the reader of the code. See [Transformation] examples.
func Permute(indices ...uint8) *Transformation {
	return &Transformation{
		typ:     permutation,
		indices: indices,
	}
}

// Transform returns a function that generates a general-purpose Transformation.
// The `depth` specifies how deep in the stack the Transformation must modify.
// The indices MUST all be less than `depth`.
//
// Note that the same indices with different depth will result in *different*
// stack outputs. See [Transformation] examples.
func Transform(depth uint8) func(indices ...uint8) *Transformation {
	return func(i ...uint8) *Transformation {
		return &Transformation{
			typ:     general,
			depth:   depth,
			indices: i,
		}
	}
}

// WithOps sets the exact opcodes that t.Bytecode() MUST return. Possible use
// cases include:
//   - Caching: worst-case performance of Permute() is n! while worst-case
//     Transform() may be higher. WithOps is linear in the number of ops.
//   - Intent signalling: if an exact sequence of opcodes is required but they
//     are opaque, the Transformation setup will inform the reader of the
//     outcome.
//
// When Bytecode() is called on the returned value, it confirms that the ops
// result in the expected transformation and then returns them verbatim.
//
// WithOps modifies t and then returns it.
func (t *Transformation) WithOps(ops ...types.OpCode) *Transformation {
	t.override = ops
	return t
}

// Bytecode returns the stack-transforming opcodes (SWAP, DUP, etc) necessary to
// achieve the transformation in the most efficient manner.
func (t *Transformation) Bytecode() ([]byte, error) {
	var sizer func() (int, error)

	switch t.typ {
	case permutation:
		sizer = t.permutationSize
	case general:
		sizer = t.generalSize
	default:
		return nil, fmt.Errorf("invalid %T.typ = %d", t, t.typ)
	}

	size, err := sizer()
	if err != nil {
		return nil, err
	}

	if len(t.override) != 0 {
		return t.overriden()
	}
	return t.bfs(size)
}

// overriden confirms that the overriding opcodes passed to t.WithOps() result
// in the expected opcode and then returns them verbatim (as bytes).
func (t *Transformation) overriden() ([]byte, error) {
	n := rootNode(uint8(t.depth))
	var err error
	for _, o := range t.override {
		n, err = n.apply(vm.OpCode(o))
		if err != nil {
			return nil, err
		}
	}
	if got, want := n, nodeFromIndices(t.indices); got != want {
		return nil, fmt.Errorf("invalid WithOps() config; transformed stack = %v; want %v", got, want)
	}

	out := make([]byte, len(t.override))
	for i, o := range t.override {
		out[i] = byte(o)
	}
	return out, nil
}

// permutationSize confirms t.indices is valid for a permutation and then
// returns the size to be passed to bfs().
func (t *Transformation) permutationSize() (int, error) {
	if n := len(t.indices); n > 16 {
		return 0, fmt.Errorf("can only permute up to 16 stack items; got %d", n)
	}
	t.depth = uint8(len(t.indices))

	set := make(map[uint8]bool)
	for _, idx := range t.indices {
		if set[idx] {
			return 0, fmt.Errorf("duplicate index %d in permutation %v", idx, t.indices)
		}
		set[idx] = true
	}

	for i := range t.indices { // explicitly not `_, i` like last loop
		if !set[uint8(i)] {
			return 0, fmt.Errorf("non-contiguous indices in permutation %v; missing %d", t.indices, i)
		}
	}
	return len(t.indices), nil
}

// generalSize confirms that t.depth and t.indices are valid for any
// transformation and then returns the size to be passed to bfs().
func (t *Transformation) generalSize() (int, error) {
	if t.depth > 16 {
		return 0, fmt.Errorf("transformation depth %d > 16", t.depth)
	}
	for _, idx := range t.indices {
		if idx >= t.depth {
			return 0, fmt.Errorf("stack index %d beyond transformation depth of %d", idx, t.depth)
		}
	}
	return int(t.depth), nil
}

// bfs performs a breadth-first search over a graph of stack-value orders,
// starting from the root, in-order node [0, size). Edges represent nodes that
// are reachable with only a single opcode.
//
// bfs should be called by the transformation-type-specific methods that first
// check for valid indices. bfs itself is, however, type-agnostic.
//
// Although POP only uses 2 gas while DUPs/SWAPs use 3, there's no need for a
// full Dijkstra implementation as changes in stack size can only be achieved by
// POP/DUP and we limit graph edges accordingly.
func (t *Transformation) bfs(size int) ([]byte, error) {
	if size == 0 || size > 16 {
		return nil, fmt.Errorf("invalid %T size %d", t, size)
	}

	root := rootNode(uint8(size))
	want := nodeFromIndices(t.indices)
	if want == root {
		return nil, nil
	}

	// An implicit graph representation that only has nodes added when enqueued
	// by the BFS.
	graph := transformationPaths{
		root: nil,
	}

	for queue := []node{root}; len(queue) > 0; {
		curr := queue[0]
		queue = queue[1:]
		currPath, ok := graph[curr]
		if !ok {
			return nil, fmt.Errorf("BUG: node %q in queue but not in graph", curr)
		}

		var edges []vm.OpCode
		delta := want.deltas(curr)
		currIndices := curr.toIndices()
		allIndices := append(want.toIndices(), currIndices...)

		for _, idx := range allIndices { // not ranging over delta, to avoid non-determinism
			switch d := delta[idx]; {
			case d == 0:
				// counts match, may need a swap but no DUP/POP

			case d > 0:
				for i, cIdx := range currIndices {
					if cIdx == idx {
						edges = append(edges, vm.DUP1+vm.OpCode(i))
						// We don't decrement delta because we can only make one
						// change per queue loop. Since it's reachable with the
						// op we've just added, there's no point following other
						// edges.
						delta[idx] = 0
						break
					}
				}

			case d < 0 && currIndices[0] == idx:
				edges = append(edges, vm.POP)
				delta[idx] = 0 // see rationale above
			}
		}

		// SWAPs are limited to len-1 because they're 1-indexed in the stack
		for i, n := 0, len(curr)-1; i < n; i++ {
			edges = append(edges, vm.SWAP1+vm.OpCode(i))
		}

		for _, op := range edges {
			next, err := curr.apply(op)
			if err != nil {
				return nil, err
			}
			// The next node has already been visited and, since this is an
			// unweighted graph, BFS ordering is sufficient for the shortest
			// path.
			if _, ok := graph[next]; ok {
				continue
			}

			nextPath := make(path, len(currPath)+1)
			copy(nextPath, currPath)
			nextPath[len(currPath)] = op

			if next == want {
				return nextPath.bytes(), nil
			}

			graph[next] = nextPath
			queue = append(queue, next)
		}
	}

	// This should never happen (famous last words!)
	return nil, fmt.Errorf("stack transformation %v not reached by BFS", t.indices)
}

// transformationPaths represent the paths to reach the specific node from the
// rootNode().
type transformationPaths map[node]path

// A node represents a slice of stack indices as a string so it can be used as a
// map key. To aid in debugging, it represents each index as a hex character,
// however this MUST NOT be relied upon to be stable.
type node string

// A path represents a set of opcodes which, if applied in order, transform the
// root node into another.
type path []vm.OpCode

// nodeFromIndices converts the indices into a node.
func nodeFromIndices(is []uint8) node {
	var s strings.Builder
	for _, i := range is {
		switch {
		case i < 10:
			s.WriteByte('0' + i)
		case i < 16:
			s.WriteByte('a' + i - 10)
		default:
			// If this happens then there's a broken invariant that should have
			// been prevented by an error-returning path. Panicking here is only
			// possible if there's a bug.
			panic(fmt.Sprintf("BUG: invalid index value %d > 15", i))
		}
	}
	return node(s.String())
}

// toIndices is the inverse of nodeToIndices().
func (n node) toIndices() []uint8 {
	is := make([]uint8, len(n))
	for i, rn := range n {
		switch {
		case '0' <= rn && rn <= '9':
			is[i] = uint8(rn - '0')
		case 'a' <= rn && rn <= 'f':
			is[i] = uint8(rn - 'a' + 10)
		default:
			// See equivalent panic in nodeFromIndices().
			panic(fmt.Sprintf("BUG: invalid %T rune %v; must be hex char", n, rn))
		}
	}
	return is
}

// rootNode returns the node representing [0, …, size).
func rootNode(size uint8) node {
	buf := make([]byte, size)
	for i := range buf {
		buf[i] = byte(i)
	}
	return nodeFromIndices(buf)
}

// deltas returns a map keyed by every rune in the union of `n` and `o`, with
// values describing the difference in counts in each. Those more prevalent in
// `n` will have positive values, more-so in `o` will have negative values, and
// runes that occur equally in both nodes will have zero as their value.
//
// The returned value indicates the number of DUPs / POPs required to convert
// `o` to `n`.
func (n node) deltas(o node) map[uint8]int {
	d := make(map[uint8]int)
	for _, i := range n.toIndices() {
		d[i]++
	}
	for _, i := range o.toIndices() {
		d[i]--
	}
	return d
}

// apply returns a *new* node equivalent to applying the opcode to n.
func (n node) apply(o vm.OpCode) (node, error) {
	switch base := o & 0xf0; {
	case o == vm.POP:
		return n[1:], nil

	case base == vm.SWAP1:
		out := make([]byte, len(n))
		copy(out, []byte(n))

		i := o - vm.SWAP1 + 1
		out[0], out[i] = out[i], out[0] // invariants in the BFS loop guarantee that these are in range

		return node(out), nil

	case base == vm.DUP1:
		out := make([]byte, len(n)+1)
		copy(out[1:], []byte(n))

		out[0] = out[o-vm.DUP1+1] // see above re invariants

		return node(out), nil

	default:
		return "", fmt.Errorf("unsupported transformation %T(%v)", o, o)
	}
}

// bytes returns p, verbatim, as bytes.
func (p path) bytes() []byte {
	out := make([]byte, len(p))
	for i, pp := range p {
		out[i] = byte(pp)
	}
	return out
}
