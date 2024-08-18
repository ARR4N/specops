package specops

import (
	"fmt"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"

	"github.com/arr4n/specops/stack"
)

func Example_helloWorld() {
	hello := []byte("Hello world")
	code := Code{
		// The compiler determines the shortest-possible PUSH<n> opcode.
		// Fn() simply reverses its arguments (a surprisingly powerful construct)!
		Fn(MSTORE, PUSH0, PUSH(hello)),
		Fn(RETURN, PUSH(32-len(hello)), PUSH(len(hello))),
	}

	compiled, err := code.Compile()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%#x\n", compiled)
	fmt.Println(string(mustRunByteCode(compiled, []byte{} /*callData*/)))

	// Output:
	// 0x6a48656c6c6f20776f726c645f52600b6015f3
	// Hello world
}

func ExampleCode_eip1167() {
	// Demonstrates verbatim recreation of EIP-1167 Minimal Proxy Contract and a
	// modern equivalent with PUSH0.

	impl := common.HexToAddress("bebebebebebebebebebebebebebebebebebebebe")
	eip1167 := Code{
		// Think of RETURNDATASIZE before DELEGATECALL as PUSH0 (the EIP predated it)
		Fn(CALLDATACOPY, RETURNDATASIZE, RETURNDATASIZE, CALLDATASIZE), // Copy calldata to memory
		RETURNDATASIZE,
		Fn( // Delegate-call the implementation, forwarding all gas, and propagating calldata
			DELEGATECALL,
			GAS,
			PUSH(impl), // Native Go values!
			RETURNDATASIZE, CALLDATASIZE, RETURNDATASIZE, RETURNDATASIZE,
		),
		stack.ExpectDepth(2), // top <suc 0> bot
		Fn(
			RETURNDATACOPY,
			DUP1,           // This could equivalently be Inverted(DUP1)==DUP4
			Inverted(DUP1), // DUP the 0 at the bottom; the compiler knows to convert this to DUP3
			RETURNDATASIZE, // Actually return-data size now
		),
		stack.ExpectDepth(2),         // <suc 0>
		SWAP1, RETURNDATASIZE, SWAP2, // <suc 0 rds>

		Fn(JUMPI, PUSH("return")),
		Fn(REVERT, stack.ExpectDepth(2)), // Compiler hint for argc

		JUMPDEST("return"),
		stack.SetDepth(2), // Required after a JUMPDEST
		RETURN,
	}

	// Using PUSH0, here is a modernised version of EIP-1167, reduced by 1 byte
	// and easy to read.
	eip1167Modern := Code{
		Fn(CALLDATACOPY, PUSH0, PUSH0, CALLDATASIZE),
		Fn(DELEGATECALL, GAS, PUSH(impl), PUSH0, CALLDATASIZE, PUSH0, PUSH0),
		stack.ExpectDepth(1), // `success`
		Fn(RETURNDATACOPY, PUSH0, PUSH0, RETURNDATASIZE),

		stack.ExpectDepth(1),  // unchanged
		PUSH0, RETURNDATASIZE, // prepare for the REVERT/RETURN; these are in "human" order because of the next SWAP
		Inverted(SWAP1), // bring `success` from the bottom
		Fn(JUMPI, PUSH("return")),

		Fn(REVERT, stack.ExpectDepth(2)),

		JUMPDEST("return"),
		Fn(RETURN, stack.SetDepth(2)),
	}

	for _, eg := range []struct {
		name string
		code Code
	}{
		{"EIP-1167", eip1167},
		{"Modernised EIP-1167", eip1167Modern},
	} {
		bytecode, err := eg.code.Compile()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%19s: %#x\n", eg.name, bytecode)
	}

	// Output:
	//
	//            EIP-1167: 0x363d3d373d3d3d363d73bebebebebebebebebebebebebebebebebebebebe5af43d82803e903d91602b57fd5bf3
	// Modernised EIP-1167: 0x365f5f375f5f365f73bebebebebebebebebebebebebebebebebebebebe5af43d5f5f3e5f3d91602a57fd5bf3
}

func ExampleCode_verbose0ageMetamorphic() {
	// Demonstrates verbatim recreation of 0age's metamorphic contract
	// constructor: https://github.com/0age/metamorphic/blob/55adac1d2487046002fc33a5dff7d669b5419a3a/contracts/MetamorphicContractFactory.sol#L55
	//
	// Using stack.Transform() automation we also see how the size could have
	// been reduced. Granted, only by a single byte, but it also saves a lot of
	// development time.

	metamorphicPrelude := Code{
		// 0age uses PC to place a 0 on the bottom of the stack and then
		// duplicates it as necessary. Using `Inverted(DUP1)` makes this
		// much easier to reason about. This is especially so when
		// refactoring as the specific DUP<N> would otherwise have to
		// change.
		Fn(
			// Although Fn() wasn't intended to be used without a
			// function-like opcode at the beginning, it sheds light on
			// what 0age was doing here: setting up all the arguments
			// for a later STATICCALL. While nested Fn()s act like
			// regular functions (see ISZERO later), sequential ones
			// have the effect of "piping" arguments to the next, which
			// may or may not use them. As the MSTORE Fn() has
			// sufficient arguments, the ones set up here are left for
			// the STATICCALL.
			//
			// Note that everything in Fn() is reversed so PCs count
			// from the right, but the rest is easier to read as it is
			// Yul-like. I'm guessing that this argument setup without
			// the call was a trick to cheaply get the PC=4 in the right
			// place.
			GAS, CALLER, PUSH(28), PC /*4*/, Inverted(DUP1) /*0*/, PUSH(32), PC,
		),
		Fn(
			MSTORE,
			Inverted(DUP1), // Compiler knows this is a DUP8 to copy the 0 from the bottom
			PUSHSelector("getImplementation()"),
		),
		// Although the inner Fn() is equivalent to a raw STATICCALL,
		// the compiler hint for the stack depth is useful (and also
		// signals the reader of the code to remember the earlier
		// setup), while placing it in Fn() makes the order more
		// readable.
		Fn(ISZERO, Fn(STATICCALL, stack.ExpectDepth(7))),
		// Recall that the return (offset, size) were set to (0,32).
		stack.ExpectDepth(2), // [0, fail?] memory:<addr>

		Fn(MLOAD, Inverted(DUP1) /*0*/), // [0, fail?, addr]
		Fn(EXTCODESIZE, DUP1),           // DUP1 as a single argument is like a stack peek
	}

	// For reference, a snippet from 0age's comments to explain the stack
	// transformation that now occurs.
	//
	// * ** get extcodesize on fourth stack item for extcodecopy **
	// * 18 3b extcodesize    [0, 0, address, size]                     <>
	// ...
	// ...
	// * 23 92 swap3          [size, 0, size, 0, 0, address]            <>

	// The stack as it currently stands, labelled top to bottom.
	const (
		size = iota
		address
		callFailed // presumably zero
		zero

		depth
	)

	metamorphic := Code{
		metamorphicPrelude,
		stack.Transform(depth)(address, zero, zero, size, callFailed, size).WithOps(
			// The exact opcodes from the original, which the compiler will
			// confirm as having the intended result.
			DUP1, SWAP4, DUP1, SWAP2, SWAP3,
		),
		stack.ExpectDepth(6),
		EXTCODECOPY,
		RETURN,
	}

	autoMetamorphic := Code{
		metamorphicPrelude,
		stack.Transform(depth)(address, zero, zero, size, callFailed, size),
		stack.ExpectDepth(6),
		EXTCODECOPY,
		RETURN,
	}

	for _, eg := range []struct {
		name string
		code Code
	}{
		{"         0age/metamorphic", metamorphic},
		{"Auto stack transformation", autoMetamorphic},
	} {
		bytecode, err := eg.code.Compile()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%19s: %#x\n", eg.name, bytecode)
	}

	// Output:
	//
	//          0age/metamorphic: 0x5860208158601c335a63aaf10f428752fa158151803b80938091923cf3
	// Auto stack transformation: 0x5860208158601c335a63aaf10f428752fa158151803b928084923cf3
}

func ExampleCode_succinct0ageMetamorphic() {
	// Identical to the other metamorphic example, but with explanatory comments
	// removed to demonstrate succinct but readable production usage.

	const zero = Inverted(DUP1) // see first opcode

	metamorphic := Code{
		// Keep a zero at the bottom of the stack
		PC,
		// Prepare a STATICCALL signature
		Fn( /*STATICCALL*/ GAS, CALLER, PUSH(28), PC /*4*/, zero, PUSH(32)),

		Fn(MSTORE, zero, PUSHSelector("getImplementation()")), // stack unchanged

		Fn(ISZERO, STATICCALL), // consumes all values except the zero
		stack.ExpectDepth(2),   // [0, fail?] <addr>

		Fn(MLOAD, zero),       // [0, fail?, addr]
		Fn(EXTCODESIZE, DUP1), // [0, fail?, addr, size]
	}

	{
		// Current stack, top to bottom
		const (
			size = iota
			address
			callFailed // presumed to be 0
			zero

			depth
		)
		metamorphic = append(
			metamorphic,
			stack.Transform(depth)(
				/*EXTCODECOPY*/ address, zero, zero, size,
				/*RETURN*/ callFailed /*0*/, size,
			).WithOps(
				// In reality we wouldn't override the ops, but let the
				// stack.Transformation find an optimal path.
				DUP1, SWAP4, DUP1, SWAP2, SWAP3,
			),
			EXTCODECOPY,
			RETURN,
		)
	}

	bytecode, err := metamorphic.Compile()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%#x", bytecode)

	// Output:
	//
	// 0x5860208158601c335a63aaf10f428752fa158151803b80938091923cf3
}

func ExampleCode_monteCarloPi() {
	// A unit circle inside a 2x2 square covers π/4 of the area. We can
	// (inefficiently) approximate π using sha3 as a source of entropy!
	//
	// Bottom of the stack will always be:
	// - loop total
	// - loops remaining
	// - hit counter (values inside the circle)
	// - constant: 1 (to use DUP instead of PUSH)
	// - constant: 1 << 128 - 1
	// - constant: 1 <<  64 - 1
	// - Entropy (hash)
	//
	// We can therefore use Inverted(DUP/SWAPn) to access them as required,
	// effectively creating variables.
	const (
		Total = Inverted(DUP1) + iota
		Limit
		Hits
		One
		Bits128
		Bits64
		Hash
	)
	const (
		SwapLimit = Limit + 16 + iota
		SwapHits
	)
	const bitPrecision = 128

	code := Code{
		PUSH(0x02b000),                         // loop total (~30M gas); kept as the denominator
		DUP1,                                   // loops remaining
		PUSH0,                                  // inside-circle count (numerator)
		PUSH(1),                                // constant-value 1
		Fn(SUB, Fn(SHL, PUSH(0x80), One), One), // 128-bit mask
		Fn(SUB, Fn(SHL, PUSH(0x40), One), One), // 64-bit mask
		stack.ExpectDepth(6),

		JUMPDEST("loop"), stack.SetDepth(6),

		Fn(KECCAK256, PUSH0, PUSH(32)),

		Fn(AND, Bits64, Hash),                    // x = lowest 64 bits
		Fn(AND, Bits64, Fn(SHR, PUSH(64), Hash)), // y = next lowest 64 bits

		Fn(GT,
			Bits128,
			Fn(ADD,
				Fn(MUL, DUP1), // y^2
				SWAP1,         // x^2 <-> y
				Fn(MUL, DUP1), // x^2
			),
		),

		Fn(SwapHits, Fn(ADD, Hits)),

		Fn(JUMPI,
			PUSH("return"),
			Fn(ISZERO, DUP1, Fn(SUB, Limit, One)), // DUP1 uses the top of the stack without consuming it
		),
		stack.ExpectDepth(9),

		SwapLimit, POP, POP,
		Fn(MSTORE, PUSH0),
		Fn(JUMP, PUSH("loop")), stack.ExpectDepth(6),

		JUMPDEST("return"), stack.SetDepth(9),
		POP, POP,
		Fn(MSTORE,
			PUSH0,
			Fn(DIV,
				Fn(SHL, PUSH(bitPrecision+2), Hits), // extra 2 to undo π/4
				Total,
			),
		),
		Fn(RETURN, PUSH0, PUSH(32)),
	}

	pi := new(big.Rat).SetFrac(
		new(big.Int).SetBytes(compileAndRun(code, []byte{})),
		new(big.Int).Lsh(big.NewInt(1), bitPrecision),
	)

	fmt.Println(pi.FloatString(2))
	// Output: 3.14
}

func ExampleCode_sqrt() {
	// This implements the same sqrt() algorithm as prb-math:
	// https://github.com/PaulRBerg/prb-math/blob/5b6279a0cf7c1b1b6a5cc96082811f7ef620cf60/src/Common.sol#L595
	// Snippets included under MIT, Copyright (c) 2023 Paul Razvan Berg
	//
	// See the Monte-Carlo π for explanation of "variables".
	const (
		Input = Inverted(DUP1) + iota
		One
		ThresholdBits
		Threshold
		xAux
		Result
		Branch
	)
	const (
		SwapInput = Input + 16 + iota
		_         // SetOne
		SetThresholdBits
		SetThreshold
		SetXAux
		SetResult
		SetBranch
	)

	// Placing stack.ExpectDepth(i/o) at the beginning/end of a Code
	// effectively turns it into a macro that can either be embedded in another
	// Code (as below) or for use in Solidity `verbatim_Xi_Yo`.
	approx := Code{
		stack.ExpectDepth(6),
		// Original:
		//
		// if (xAux >= 2 ** 128) {
		//   xAux >>= 128;
		//   result <<= 64;
		// }
		// if (xAux >= 2 ** 64) {
		// ...
		//
		Fn(GT, xAux, Threshold), // Branch

		Fn(SetXAux,
			Fn(SHR,
				Fn(MUL, ThresholdBits, Branch),
				xAux,
			),
		), POP, // old value; TODO: improve this by using a SWAP instead of a DUP inside the Fn()

		Fn(SetThresholdBits,
			Fn(SHR, One, ThresholdBits),
		), POP,

		Fn(SetThreshold,
			Fn(SUB, Fn(SHL, ThresholdBits, One), One),
		), POP,

		Fn(SetResult,
			Fn(SHL,
				Fn(MUL, ThresholdBits, Branch),
				Result,
			),
		), POP,

		POP, // Branch
		stack.ExpectDepth(6),
	}

	// Single round of Newton–Raphson
	newton := Code{
		stack.ExpectDepth(6),
		// Original: result = (result + x / result) >> 1;
		Fn(SetResult,
			Fn(SHR,
				One,
				Fn(ADD,
					Result,
					Fn(DIV, Input, Result),
				),
			),
		), POP,
		stack.ExpectDepth(6),
	}

	sqrt := Code{
		stack.ExpectDepth(1), // Input
		PUSH(1),              // One
		PUSH(128),            // ThresholdBits
		Fn(SUB, Fn(SHL, ThresholdBits, One), One), // Threshold
		Input, // xAux := Input
		One,   // Result
		stack.ExpectDepth(6),

		approx, approx, approx, approx, approx, approx, approx,
		stack.ExpectDepth(6),
		newton, newton, newton, newton, newton, newton, newton,
	}

	code := Code{
		Fn(CALLDATALOAD, PUSH0),
		sqrt,
		Fn(MSTORE, PUSH0),
		Fn(RETURN, PUSH0, PUSH(32)),
	}

	root := new(uint256.Int) // can we get this back? ;)
	if err := root.SetFromHex("0xDecafC0ffeeBad15DeadC0deCafe"); err != nil {
		log.Fatal(err)
	}
	callData := new(uint256.Int).Mul(root, root).Bytes32()

	result := new(uint256.Int).SetBytes(
		compileAndRun(code, callData),
	)

	fmt.Println("    In:", root.Hex())
	fmt.Println("Result:", result.Hex())
	fmt.Println(" Equal:", root.Eq(result))

	// Output:
	// 	   In: 0xdecafc0ffeebad15deadc0decafe
	// Result: 0xdecafc0ffeebad15deadc0decafe
	//  Equal: true
}

func ExamplePUSH_jumpTable() {
	// This is a highly optimised factorial function, implementing one of the
	// Curta gas-golfing (https://www.curta.wtf/golf/2) solutions by philogy.eth
	// https://basescan.org/address/0x550d8df432706504b550c7cf93660cd362d7f95c

	prod := func(start, end uint64) uint64 {
		x := end
		for i := start; i < end; i++ {
			x *= i
		}
		return x
	}

	rangeMuls := Code{
		JUMPDEST("49:54"), stack.SetDepth(2),
		Fn(MUL,
			PUSH(prod(49, 54)),
		),

		JUMPDEST("43:48"), stack.SetDepth(2),
		Fn(MUL,
			PUSH(prod(43, 48)),
		),

		JUMPDEST("37:42"), stack.SetDepth(2),
		Fn(MUL,
			PUSH(prod(37, 42)),
		),

		JUMPDEST("31:36"), stack.SetDepth(2),
		Fn(MUL,
			PUSH(prod(31, 36)),
		),

		JUMPDEST("25:30"), stack.SetDepth(2),
		Fn(MUL,
			PUSH(prod(25, 30)),
		),

		JUMPDEST("19:24"), stack.SetDepth(2),
		Fn(MUL,
			PUSH(prod(19, 24)),
		),

		JUMPDEST("13:18"), stack.SetDepth(2),
		Fn(MUL,
			PUSH(prod(13, 18)),
		),

		JUMPDEST("7:12"), stack.SetDepth(2),
		Fn(MUL,
			PUSH(prod(7, 12)),
		),

		JUMPDEST("1:6"), stack.SetDepth(2),
		Fn(MUL,
			PUSH(prod(1, 6)),
		),

		JUMPDEST("no-range-mul"), stack.SetDepth(2),
	}
	ranges := []string{
		"no-range-mul",
		"1:6",
		"7:12",
		"13:18",
		"19:24",
		"25:30",
		"31:36",
		"37:42",
		"43:48",
		"49:54",
	}

	const Input = Inverted(DUP1) // always bottom of the stack

	remainderMuls := Code{
		JUMPDEST("sub4"), stack.SetDepth(2),
		Fn(MUL,
			Fn(SUB, Input, PUSH(4)),
		),

		JUMPDEST("sub3"), stack.SetDepth(2),
		Fn(MUL,
			Fn(SUB, Input, PUSH(3)),
		),

		JUMPDEST("sub2"), stack.SetDepth(2),
		Fn(MUL,
			Fn(SUB, Input, PUSH(2)),
		),

		JUMPDEST("sub1"), stack.SetDepth(2),
		Fn(MUL,
			Fn(SUB, Input, PUSH(1)),
		),

		JUMPDEST("sub0"), stack.SetDepth(2),
		MUL, /* result * input */
		stack.ExpectDepth(1),

		JUMPDEST("no-remainder-mul"), stack.SetDepth(2),
	}
	remainders := []string{
		"no-remainder-mul",
		"sub0",
		"sub1",
		"sub2",
		"sub3",
		"sub4",
	}

	const divisor = 6

	code := Code{
		PUSH(4),
		CALLDATALOAD,
		PUSH(1), // Result

		Fn(JUMPI,
			Fn(BYTE,
				Fn(ADD,
					Fn(DIV, Input, PUSH(divisor)),
					PUSH(32-len(ranges)),
				),
				PUSH(ranges),
			),
			Fn(LT, Input, PUSH(58)),
		),

		RETURNDATASIZE,
		RETURNDATASIZE,
		REVERT,

		rangeMuls,

		Fn(JUMP,
			Fn(BYTE,
				Fn(ADD,
					Fn(MOD, Input, PUSH(divisor)),
					PUSH(32-len(remainders)),
				),
				PUSH(remainders),
			),
		),

		remainderMuls,

		Fn(MSTORE, RETURNDATASIZE),
		Fn(RETURN, RETURNDATASIZE, MSIZE),
	}

	got, err := code.Compile()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%#x", got)

	// Output: 0x6004356001603a8210695d58524c453e37302820601660068504011a573d3dfd5b64045461b590025b64020ea2db80025b63e11fed20025b6353971500025b63197b6830025b6305c6b740025b62cbf340025b620a26c0025b6102d0025b658886807a746e601a60068406011a565b60048203025b60038203025b60028203025b60018203025b025b3d52593df3
}

func ExampleLabel() {
	const size = Inverted(DUP1)

	dataTable := Code{
		PUSHSize("data", "end"), // calculated during compilation

		Fn(CODECOPY, PUSH0, PUSH("data"), size),
		Fn(RETURN, PUSH0 /* size already on stack */),

		Label("data"), // not compiled into anything
		Raw("hello world"),
		Label("end"),
	}

	fmt.Println(string(compileAndRun(dataTable, []byte{})))

	// Output: hello world
}

func compileAndRun[T interface{ []byte | [32]byte }](code Code, callData T) []byte {
	var slice []byte
	switch c := any(callData).(type) {
	case []byte:
		slice = c
	case [32]byte:
		slice = c[:]
	}

	got, err := code.Run(slice)
	if err != nil {
		log.Fatal(err)
	}
	return got.ReturnData
}
