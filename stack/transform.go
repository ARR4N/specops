package stack

import (
	"errors"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/core/vm"
)

type xFormType int

const (
	unknownXform xFormType = iota
	permutation
)

// A Transformation transforms the stack by modifying its order, growing, and/or
// shrinking it.
type Transformation struct {
	typ     xFormType
	indices []uint8
	cache   string
}

// Permute returns a Transformation that permutes the order of the stack. The
// indices MUST be a contiguous set of distinct values [0,n) in any order.
func Permute(indices ...uint8) *Transformation {
	return &Transformation{
		typ:     permutation,
		indices: indices,
	}
}

// Bytecode returns the stack-transforming opcodes (SWAP, DUP, etc) necessary to
// achieve the transformation in the most efficient manner.
func (t *Transformation) Bytecode() ([]byte, error) {
	if t.cache != "" {
		return t.cached()
	}

	switch t.typ {
	case permutation:
		return t.permute()
	default:
		return nil, fmt.Errorf("invalid %T.typ = %d", t, t.typ)
	}
}

func (t *Transformation) cached() ([]byte, error) {
	return nil, errors.New("cached transformations unimplemented")
}

// permute checks that t.indices is valid for a permutation and then returns
// t.bfs().
func (t *Transformation) permute() ([]byte, error) {
	if n := len(t.indices); n > 16 {
		return nil, fmt.Errorf("can only permute up to 16 stack items; got %d", n)
	}

	set := make(map[uint8]bool)
	for _, idx := range t.indices {
		if set[idx] {
			return nil, fmt.Errorf("duplicate index %d in permutation %v", idx, t.indices)
		}
		set[idx] = true
	}

	for i := range t.indices { // explicitly not `_, i` like last loop
		if !set[uint8(i)] {
			return nil, fmt.Errorf("non-contiguous indices in permutation %v; missing %d", t.indices, i)
		}
	}
	return t.bfs(len(t.indices))
}

// bfs performs a breadth-first search over a graph of stack-value orders,
// starting from the root, in-order node [0, size). Edges represent nodes that
// are reachable with only a single opcode.
//
// bfs should be called by the transformation-type-specific methods that first
// check for valid indices. bfs itself is, however, type-agnostic.
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

		// SWAPs are limited to n-1 because they're 1-indexed in the stack
		for i, n := 0, len(t.indices)-1; i < n; i++ {
			op := vm.SWAP1 + vm.OpCode(i)
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

// rootNode returns the node representing [0, …, size).
func rootNode(size uint8) node {
	buf := make([]byte, size)
	for i := range buf {
		buf[i] = byte(i)
	}
	return nodeFromIndices(buf)
}

// apply returns a *new* node equivalent to applying the opcode to n.
func (n node) apply(o vm.OpCode) (node, error) {
	switch base := o & 0xf0; {
	case base == vm.SWAP1:
		out := make([]byte, len(n))
		copy(out, []byte(n))

		i := o - vm.SWAP1 + 1
		out[0], out[i] = out[i], out[0] // invariants in the BFS loop guarantee that these are in range

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
