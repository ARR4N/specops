package jump_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	. "github.com/solidifylabs/specops"
	"github.com/solidifylabs/specops/jump"
	"github.com/solidifylabs/specops/stack"
)

func TestPushTab(t *testing.T) {
	code := Code{
		PUSH0,
		JUMPDEST("a"), stack.SetDepth(0), // PC = 1
		PUSH0, PUSH0, PUSH0, // PC = 2,3,4
		JUMPDEST("b"), stack.SetDepth(0), // PC = 5
		JUMPDEST("c"), stack.SetDepth(0), // PC = 6

		Fn(MSTORE, PUSH0, PUSH(jump.Table{"a", "b", "c"})),
		Fn(RETURN, PUSH0, MSIZE),
	}

	got, err := code.Run(nil)
	if err != nil {
		t.Fatalf("%T.Run() error %v", code, err)
	}

	var want [32]byte
	want[29] = 1
	want[30] = 5
	want[31] = 6

	if diff := cmp.Diff(want[:], got); diff != "" {
		t.Errorf("%T.Run() diff (-want +got):\n%s", code, diff)
	}
}
