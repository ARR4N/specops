package jump_test

import (
	"testing"

	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/google/go-cmp/cmp"
	. "github.com/solidifylabs/specops"
	"github.com/solidifylabs/specops/jump"
	"github.com/solidifylabs/specops/stack"
)

func TestPushTable(t *testing.T) {
	code := Code{
		JUMPDEST("0"), stack.SetDepth(0),
		JUMPDEST("1"), stack.SetDepth(0),

		make(Raw, 18), // ...19

		JUMPDEST("20"), stack.SetDepth(0),
		PUSH(jump.Table{"1", "20", "0"}), // 21, 22, 23, 24
		JUMPDEST("25"), stack.SetDepth(0),

		make(Raw, 25), // ...50

		JUMPDEST("51"), stack.SetDepth(0),
		PUSH("51"),  // 52, 53
		PUSH("100"), // 54, 55 (although forward-looking, only 1 byte)

		make(Raw, 5), // ...60

		PUSH(jump.Table{"20", "25", "100"}), // 61, 62, 63, 64

		make(Raw, 35), // ...99

		JUMPDEST("100"), stack.SetDepth(0),
		PUSH("255"),                   // 101, 102
		PUSH(jump.Table{"255"}),       // 103,104
		PUSH(jump.Table{"255", "51"}), // 105, 106, 107
		PUSH("261"),                   // 108, [109, 110]
		PUSH(jump.Table{"261"}),       // 111, [112, 113]

		// Although all must be the same size, leading zeroes are still stripped
		PUSH(jump.Table{"51", "261"}), // 114, [115], [116, 117]
		PUSH(jump.Table{"261", "51"}), // 118, [119, 120], [121, 122]

		make(Raw, 132), // ...254

		JUMPDEST("255"), stack.SetDepth(0),
		// Ideally the test would have the next JUMPDEST at 256, but that
		// actually allows all of the PUSHes to be compressed sufficiently to
		// all fit in single bytes.
		make(Raw, 5),
		JUMPDEST("261"), stack.SetDepth(0),
	}

	want := make([]vm.OpCode, 262)
	for _, i := range []int{0, 1, 20, 25, 51, 100, 255, 261} {
		want[i] = vm.JUMPDEST
	}

	want[21] = vm.PUSH3
	want[22] = 1
	want[23] = 20
	want[24] = 0

	want[52] = vm.PUSH1
	want[53] = 51

	want[54] = vm.PUSH1
	want[55] = 100

	want[61] = vm.PUSH3
	want[62] = 20
	want[63] = 25
	want[64] = 100

	want[101] = vm.PUSH1
	want[102] = 255

	want[103] = vm.PUSH1
	want[104] = 255

	want[105] = vm.PUSH2
	want[106] = 255
	want[107] = 51

	want[108] = vm.PUSH2
	want[109] = 261 >> 8
	want[110] = 261 & 0xff

	want[111] = vm.PUSH2
	want[112] = 261 >> 8
	want[113] = 261 & 0xff

	want[114] = vm.PUSH3
	// Leading zero stripped
	want[115] = 51 & 0xff
	want[116] = 261 >> 8
	want[117] = 261 & 0xff

	want[118] = vm.PUSH4
	// As above but no leading zero to strip
	want[119] = 261 >> 8
	want[120] = 261 & 0xff
	want[121] = 51 >> 8
	want[122] = 51 & 0xff

	got, err := code.Compile()
	if err != nil {
		t.Fatalf("%T.Compile() error %v", code, err)
	}

	if diff := cmp.Diff(asBytes(want), got); diff != "" {
		t.Errorf("%T.Compile() diff (-want +got):\n%s", code, diff)
	}

	t.Logf(" got: %d %#x", len(got), got)
	// t.Logf("want: %d %#x", len(want), want)
}

func asBytes(ops []vm.OpCode) []byte {
	b := make([]byte, len(ops))
	for i, o := range ops {
		b[i] = byte(o)
	}
	return b
}
