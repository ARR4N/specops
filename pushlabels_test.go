package specops

import (
	"testing"

	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/google/go-cmp/cmp"
	"github.com/solidifylabs/specops/stack"
	"github.com/solidifylabs/specops/types"
)

func TestPUSHLabels(t *testing.T) {
	const (
		start Label = "start"
		adj0  Label = "Adjacent_0"
		adj1  Label = "Adjacent_1"
	)

	code := Code{
		start,
		JUMPDEST("0"), stack.SetDepth(0),
		JUMPDEST("1"), stack.SetDepth(0),

		make(Raw, 8),          // ...9
		PUSHSize("51", "100"), // 10, 11
		make(Raw, 8),          // ...19

		JUMPDEST("20"), stack.SetDepth(0),
		PUSH([]string{"1", "20", "0"}), // 21, 22, 23, 24
		JUMPDEST("25"), stack.SetDepth(0),

		make(Raw, 25), // ...50

		JUMPDEST("51"), stack.SetDepth(0),
		PUSH("51"),  // 52, 53
		PUSH("100"), // 54, 55 (although forward-looking, only 1 byte)

		make(Raw, 5), // ...60

		PUSH([]string{"20", "25", "100"}), // 61, 62, 63, 64

		make(Raw, 25),        // ...89
		PUSHSize(adj0, adj1), // 90; PUSH0 because they're adjacent
		make(Raw, 9),         // ...99

		JUMPDEST("100"), stack.SetDepth(0),
		PUSH("255"),                 // 101, 102
		PUSH([]string{"255"}),       // 103,104
		PUSH([]string{"255", "51"}), // 105, 106, 107
		PUSH("261"),                 // 108, [109, 110]
		PUSH([]string{"261"}),       // 111, [112, 113]

		// Although all must be the same size, leading zeroes are still stripped
		PUSH([]string{"51", "261"}), // 114, [115], [116, 117]
		PUSH([]string{"261", "51"}), // 118, [119, 120], [121, 122]

		make(Raw, 132), // ...254

		JUMPDEST("255"), stack.SetDepth(0),
		// Ideally the test would have the next JUMPDEST at 256, but that
		// actually allows all of the PUSHes to be compressed sufficiently to
		// all fit in single bytes.
		make(Raw, 5),
		JUMPDEST("261"), stack.SetDepth(0),

		adj0, adj1,

		PUSHSize(start, "261"), // 262, 263, 264
	}

	want := make([]vm.OpCode, 265)
	for _, i := range []int{0, 1, 20, 25, 51, 100, 255, 261} {
		want[i] = vm.JUMPDEST
	}

	want[10] = vm.PUSH1
	want[11] = 100 - 51

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

	want[90] = vm.PUSH0

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

	want[262] = vm.PUSH2
	want[263] = 261 >> 8
	want[264] = 261 & 0xff

	got, err := code.Compile()
	if err != nil {
		t.Fatalf("%T.Compile() error %v", code, err)
	}

	if diff := cmp.Diff(asBytes(want...), got); diff != "" {
		t.Errorf("%T.Compile() diff (-want +got):\n%s", code, diff)
	}

	t.Logf(" got: %d %#x", len(got), got)
	// t.Logf("want: %d %#x", len(want), want)
}

func TestPUSHNoLabels(t *testing.T) {
	// Arbitrary boundaries to surround the empty push; MUST have no effect on
	// the stack otherwise compilation will fail.
	const (
		before = MSIZE
		after  = GAS
	)

	code := Code{
		before,
		PUSH([]JUMPDEST{}),
		PUSH([]Label{}),
		PUSH([]string{}),
		after,
	}
	want := asBytes(before, after)

	t.Logf("%T = {%s, PUSH([]JUMPDEST{}, []Label{}, []string), %s}", code, before, after)

	got, err := code.Compile()
	if err != nil {
		t.Fatalf("%T.Compile() error %v", code, err)
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("%T.Compile() diff (-want +got):\n%s", code, diff)
	}
}

type opCode interface {
	vm.OpCode | types.OpCode
}

func asBytes[T opCode](ops ...T) []byte {
	b := make([]byte, len(ops))
	for i, o := range ops {
		b[i] = byte(o)
	}
	return b
}
