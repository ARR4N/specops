// Package stack_test avoids a circular dependency between the specops and stack
// packages.
package stack_test

import (
	"fmt"
	"log"
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/solidifylabs/specops"
	"github.com/solidifylabs/specops/evmdebug"
	"github.com/solidifylabs/specops/stack"
)

func ExampleTransformation() {
	egs := []struct {
		desc  string
		xform *stack.Transformation
	}{
		{
			desc:  "Permute",
			xform: stack.Permute(2, 0, 3, 1),
		},
		{
			desc: "Permute via Transform",
			// Although this is equivalent to Permute(), its verbose, intent
			// isn't clear, and there are no checks that it's a valid
			// permutation.
			xform: stack.Transform(4)(2, 0, 3, 1),
		},
		{
			desc: "Transform same depth",
			// Guaranteed *not* to have POPs because all stack items in [0,5)
			// are used.
			xform: stack.Transform(5)(4, 0, 2, 2, 3, 1),
		},
		{
			desc: "Transform greater depth",
			// Guaranteed to have POPs because, although the same indices as
			// above, a greater stack depth is being transformed. Stack items
			// {5,6} need to be removed.
			xform: stack.Transform(7)(4, 0, 2, 2, 3, 1),
		},
		{
			desc:  "Noop Transform",
			xform: stack.Transform(2)(0, 1),
		},
		{
			desc:  "Noop Permute",
			xform: stack.Permute(0, 1, 2, 3, 4, 5),
		},
	}

	for _, eg := range egs {
		bytecode, err := eg.xform.Bytecode()
		if err != nil {
			log.Fatalf("%s error %v", eg.desc, err)
		}

		ops := make([]vm.OpCode, len(bytecode))
		for i, b := range bytecode {
			ops[i] = vm.OpCode(b)
		}

		fmt.Println(eg.desc, ops)
	}

	// Output:
	// Permute [SWAP1 SWAP3 SWAP2]
	// Permute via Transform [SWAP1 SWAP3 SWAP2]
	// Transform same depth [DUP3 SWAP2 SWAP5]
	// Transform greater depth [SWAP2 SWAP3 SWAP5 POP SWAP5 POP DUP2 SWAP3]
	// Noop Transform []
	// Noop Permute []
}

func intPtr(x int) *int {
	return &x
}

func TestTransformations(t *testing.T) {
	type test struct {
		name string
		fn   func(...uint8) *stack.Transformation
		// Setup of existing stack [0 … depth)
		// For Permute(), depth == len(indices)
		// Otherwise Transform(depth)
		depth        int
		indices      []uint8
		wantNumSteps *int // don't know when fuzzing so only test if non-nil
	}

	tests := []test{
		{
			name:         "noop Permute",
			fn:           stack.Permute,
			depth:        4,
			indices:      []uint8{0, 1, 2, 3},
			wantNumSteps: intPtr(0),
		},
		{
			name:         "single-SWAP Permute",
			fn:           stack.Permute,
			depth:        8,
			indices:      []uint8{7, 1, 2, 3, 4, 5, 6, 0},
			wantNumSteps: intPtr(1),
		},
		{
			name:         "single-SWAP Permute",
			fn:           stack.Permute,
			depth:        7,
			indices:      []uint8{4, 1, 2, 3, 0, 5, 6},
			wantNumSteps: intPtr(1),
		},
		{
			name:    "Permute",
			fn:      stack.Permute,
			depth:   4,
			indices: []uint8{2, 1, 0, 3},
		},
		{
			name:    "Permute",
			fn:      stack.Permute,
			depth:   4,
			indices: []uint8{3, 2, 1, 0},
		},
		{
			name:    "Permute",
			fn:      stack.Permute,
			depth:   7,
			indices: []uint8{5, 0, 6, 3, 4, 2, 1},
		},
		{
			name:         "single-POP Transform",
			fn:           stack.Transform(5),
			depth:        5,
			indices:      []uint8{1, 2, 3, 4},
			wantNumSteps: intPtr(1),
		},
		{
			name:         "POP-all Transform",
			fn:           stack.Transform(7),
			depth:        7,
			indices:      []uint8{},
			wantNumSteps: intPtr(7),
		},
		{
			name:         "single-DUP Transform",
			fn:           stack.Transform(5),
			depth:        5,
			indices:      []uint8{3, 0, 1, 2, 3, 4},
			wantNumSteps: intPtr(1),
		},
		{
			name:    "Transform",
			fn:      stack.Transform(5),
			depth:   5,
			indices: []uint8{1, 4, 0, 0, 1, 2, 0, 3},
		},
		{
			name:    "Transform with POP",
			fn:      stack.Transform(4),
			depth:   4,
			indices: []uint8{1, 3, 0, 3},
		},
		{
			name:    "Transform with extra POPs",
			fn:      stack.Transform(6),
			depth:   6,
			indices: []uint8{1, 3, 0, 3},
		},
	}

	rng := rand.New(rand.NewSource(42))

	for i := 0; i < 20; i++ {
		in := []uint8{0, 1, 2, 3, 4, 5, 6, 7}
		rng.Shuffle(len(in), func(i, j int) {
			in[i], in[j] = in[j], in[i]
		})

		tests = append(tests, test{
			name:    "Fuzz Permute",
			fn:      stack.Permute,
			depth:   len(in),
			indices: in,
		})
	}

	for i := 0; i < 50; i++ {
		const depth = 5
		indices := make([]uint8, rng.Intn(10))
		for i := range indices {
			indices[i] = uint8(rng.Intn(depth))
		}

		tests = append(tests, test{
			name:    "Fuzz Transform",
			fn:      stack.Transform(depth),
			depth:   depth,
			indices: indices,
		})
	}

	for _, tt := range tests {
		tt := tt // for use with t.Parallel()
		t.Run(fmt.Sprintf("%s n=%d %v", tt.name, tt.depth, tt.indices), func(t *testing.T) {
			t.Parallel()

			var code specops.Code
			for i := tt.depth; i > 0; i-- {
				code = append(code, specops.PUSH(i-1)) // {0 … depth-1} top to bottom
			}

			xform := tt.fn(tt.indices...)
			code = append(code, xform)

			steps, err := xform.Bytecode()
			if err != nil {
				t.Fatalf("Bad test setup; Permute/Transform(%v).Bytecode() error %v", tt.indices, err)
			}
			for _, s := range steps {
				t.Log(vm.OpCode(s))
			}
			if got := len(steps); tt.wantNumSteps != nil && got != *tt.wantNumSteps {
				t.Errorf("Permute/Transform(%v) got %d swaps; want %d", tt.indices, got, *tt.wantNumSteps)
			}

			dbg, _, err := code.StartDebugging(nil)
			if err != nil {
				t.Fatalf("%T.StartDebugging(nil) error %v", code, err)
			}
			defer dbg.FastForward()

			for i := 0; i < tt.depth; i++ {
				dbg.Step()
			}
			inOrder := make([]uint8, tt.depth)
			for i := range inOrder {
				inOrder[i] = uint8(i)
			}
			t.Run("after PUSHing indices in order", stackTest(dbg, inOrder))

			for i := 0; i < len(steps); i++ {
				dbg.Step()
			}
			t.Run("after SWAPing based on Permute()", stackTest(dbg, tt.indices))
		})
	}
}

// stackTest returns a test function that checks the current stack values.
func stackTest(dbg *evmdebug.Debugger, want8 []uint8) func(*testing.T) {
	return func(t *testing.T) {
		t.Helper()
		st := dbg.State()
		if st.Err != nil {
			t.Fatalf("%T.State().Err = %v; want nil", dbg, st.Err)
		}

		var got []uint64
		stack := st.ScopeContext.Stack
		for i, n := 0, len(stack.Data()); i < n; i++ {
			g := stack.Back(i)
			if !g.IsUint64() {
				t.Fatalf("%T.State().ScopeContext.Stack.Data()[%d] not representable as uint64", dbg, i)
			}
			got = append(got, g.Uint64())
		}

		want := make([]uint64, len(want8))
		for i, w := range want8 {
			want[i] = uint64(w)
		}
		if diff := cmp.Diff(want, got, cmpopts.EquateEmpty()); diff != "" {
			t.Errorf("Stack [top to bottom] diff (-want +got):\n%s", diff)
		}
	}
}
