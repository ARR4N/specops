// Package stack_test avoids a circular dependency between the specops and stack
// packages.
package stack_test

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/google/go-cmp/cmp"
	"github.com/solidifylabs/specops"
	"github.com/solidifylabs/specops/evmdebug"
	"github.com/solidifylabs/specops/stack"
)

func intPtr(x int) *int {
	return &x
}

func TestPermute(t *testing.T) {
	type test struct {
		indices      []uint8
		wantNumSwaps *int // don't know when fuzzing so only test if non-nil
	}

	tests := []test{
		{
			indices:      []uint8{0, 1, 2, 3},
			wantNumSwaps: intPtr(0),
		},
		{
			indices:      []uint8{7, 1, 2, 3, 4, 5, 6, 0},
			wantNumSwaps: intPtr(1),
		},
		{
			indices:      []uint8{4, 1, 2, 3, 0, 5, 6},
			wantNumSwaps: intPtr(1),
		},
		{
			indices: []uint8{2, 1, 0, 3},
		},
		{
			indices: []uint8{3, 2, 1, 0},
		},
		{
			indices: []uint8{5, 0, 6, 3, 4, 2, 1},
		},
	}

	rng := rand.New(rand.NewSource(42))
	for i := 0; i < 20; i++ {
		in := []uint8{0, 1, 2, 3, 4, 5, 6, 7}
		rng.Shuffle(len(in), func(i, j int) {
			in[i], in[j] = in[j], in[i]
		})
		tests = append(tests, test{indices: in})
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v", tt.indices), func(t *testing.T) {
			var code specops.Code
			for i := range tt.indices { // explicitly not _, i
				code = append(code, specops.PUSH(len(tt.indices)-i-1)) // {0 … n-1} top to bottom
			}

			perm := stack.Permute(tt.indices...)
			code = append(code, perm)

			swaps, err := perm.Bytecode()
			if err != nil {
				t.Fatalf("Bad test setup; Permute(%v).Bytecode() error %v", tt.indices, err)
			}
			for _, s := range swaps {
				t.Log(vm.OpCode(s))
			}
			if got := len(swaps); tt.wantNumSwaps != nil && got != *tt.wantNumSwaps {
				t.Errorf("Permute(%v) got %d swaps; want %d", tt.indices, got, *tt.wantNumSwaps)
			}

			dbg, _, err := code.StartDebugging(nil)
			if err != nil {
				t.Fatalf("%T.StartDebugging(nil) error %v", code, err)
			}
			defer dbg.FastForward()

			for i := 0; i < len(tt.indices); i++ {
				dbg.Step()
			}
			inOrder := make([]uint8, len(tt.indices))
			for i := range inOrder {
				inOrder[i] = uint8(i)
			}
			t.Run("after PUSHing indices in order", wantStackTest(dbg, inOrder))

			for i := 0; i < len(swaps); i++ {
				dbg.Step()
			}
			t.Run("after SWAPing based on Permute()", wantStackTest(dbg, tt.indices))
		})
	}
}

// wantStackTest returns a test function that checks the current stack values.
func wantStackTest(dbg *evmdebug.Debugger, want8 []uint8) func(*testing.T) {
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
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("Stack [top to bottom] diff (-want +got):\n%s", diff)
		}
	}
}
