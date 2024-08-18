# SpecOps [![Go](https://github.com/arr4n/specops/actions/workflows/go.yml/badge.svg)](https://github.com/arr4n/specops/actions/workflows/go.yml) [![Go Reference](https://pkg.go.dev/badge/github.com/arr4n/specops.svg)](https://pkg.go.dev/github.com/arr4n/specops)

**`specops` is a low-level, domain-specific language and compiler for crafting [Ethereum VM](https://ethereum.org/en/developers/docs/evm) bytecode. The project also includes a CLI with code execution and terminal-based debugger.**

## _special_ opcodes

Writing bytecode is hard. Tracking stack items is difficult enough, made worse by refactoring that renders every `DUP` and `SWAP` off-by-X.
[Reverse Polish Notation](https://en.wikipedia.org/wiki/Reverse_Polish_notation) may be suited to stack-based programming, but it's unintuitive when context-switching from Solidity.

There's always a temptation to give up and use a higher-level language with all of its conveniences, but that defeats the point.
What if we could maintain full control of the opcode placement, but with syntactic sugar to help the medicine go down?

*Special* opcodes provide just that.
Some of them are interpreted by the compiler, converting them into [regular](https://pkg.go.dev/github.com/ethereum/go-ethereum/core/vm#OpCode) equivalents, while others are simply compiler hints that leave the resulting bytecode unchanged.

## Getting started

See the [`getting-started/`](https://github.com/arr4n/specops/tree/main/getting-started) directory for creating your first SpecOps code. Also check out the [examples](#other-examples) and the [documentation](#documentation).

### Do I have to learn Go?

No.

There's more about this in the `getting-started/` README, including the rationale for a Go-based DSL.

## Features

New features will be prioritised based on demand. If there's something you'd like included, please file an Issue.

- [x] `JUMPDEST` labels (absolute)
- [ ] `JUMPDEST` labels (relative to `PC`)
- [x] `PUSH(JUMPDEST)` by label with minimal bytes (1 or 2)
- [x] `Label` tags; like `JUMPDEST` but don't add to code
- [x] Push multiple, concatenated `JUMPDEST` / `Label` tags as one word
- [x] `PUSHSize(T,T)` pushes `Label` and/or `JUMPDEST` distance
- [x] Function-like syntax (i.e. Reverse Polish Notation is optional)
- [x] Inverted `DUP`/`SWAP` special opcodes from "bottom" of stack (a.k.a. pseudo-variables)
- [x] `PUSH<T>` for native Go types
- [X] `PUSH(v)` length detection
- [x] Macros
- [x] Compiler-state assertions (e.g. expected stack depth)
- [x] Automated optimal (least-gas) stack transformations
  - [x] Permutations (`SWAP`-only transforms)
  - [x] General-purpose (combined `DUP` + `SWAP` + `POP`)
  - [x] Caching of search for optimal route
- [ ] Standalone compiler
- [x] In-process EVM execution (geth)
  - [x] Full control of configuration (e.g. `params.ChainConfig` and `vm.Config`)
  - [x] State preloading (e.g. other contracts to call) and inspection (e.g. `SSTORE` testing)
  - [x] Message overrides (caller and value)
- [x] Debugger
  * [x] Stepping
  * [ ] Breakpoints
  * [x] Programmatic inspection (e.g. native Go tests at opcode resolution)
    * [x] Memory
    * [x] Stack
  * [x] User interface
- [ ] Source mapping
- [ ] Coverage analysis
- [ ] Fork testing with RPC URL

### Documentation

The [`specops` Go
documentation](https://pkg.go.dev/github.com/arr4n/specops) covers all
functionality.

## Examples

### Hello world

To run this example `Code` block with the SpecOps CLI, see the `getting-started/` directory.

```go
import . github.com/arr4n/specops

…

hello := []byte("Hello world")
code := Code{
    // The compiler determines the shortest-possible PUSH<n> opcode.
    // Fn() simply reverses its arguments (a surprisingly powerful construct)!
    Fn(MSTORE, PUSH0, PUSH(hello)),
    Fn(RETURN, PUSH(32-len(hello)), PUSH(len(hello))),
}

// ----- COMPILE -----
bytecode, err := code.Compile()
// ...

// ----- EXECUTE -----

result, err := code.Run(nil /*callData*/ /*, [runopts.Options]...*/)
// ...

// ----- DEBUG (Programmatic) -----
//
// ***** See below for the debugger's terminal UI *****
//

dbg, results := code.StartDebugging(nil /*callData*/ /*, Options...*/)
defer dbg.FastForward() // best practice to avoid resource leaks

state := dbg.State() // is updated on calls to Step() / FastForward()

for !dbg.Done() {
  dbg.Step()
  fmt.Println("Peek-a-boo", state.ScopeContext.Stack().Back(0))
}

result, err := results()
//...
```

### Other examples

- Verbatim reimplementation of well-known contracts
  * [EIP-1167 Minimal Proxy](https://github.com/arr4n/specops/blob/b03a75d713bffaec8cbf4b60f235f783e11bbc82/examples_test.go#L36) ([original](https://eips.ethereum.org/EIPS/eip-1167#specification))
  * 0age/metamorphic ([original](https://github.com/0age/metamorphic/blob/55adac1d2487046002fc33a5dff7d669b5419a3a/contracts/MetamorphicContractFactory.sol#L55))
    - [Verbose version](https://github.com/arr4n/specops/blob/b03a75d713bffaec8cbf4b60f235f783e11bbc82/examples_test.go#L108) with explanation of SpecOps functionality + an alternative with automated stack transformation (saves a whole 3 gas!)
    - [Succinct version](https://github.com/arr4n/specops/blob/b03a75d713bffaec8cbf4b60f235f783e11bbc82/examples_test.go#L217) as if writing production code
- [Monte Carlo approximation of pi](https://github.com/arr4n/specops/blob/41efe932c9a85e45ce705b231577447e6c944487/examples_test.go#L158)
- [`sqrt()`](https://github.com/arr4n/specops/blob/41efe932c9a85e45ce705b231577447e6c944487/examples_test.go#L246) as seen ~~on TV~~ in `prb-math` ([original](https://github.com/PaulRBerg/prb-math/blob/5b6279a0cf7c1b1b6a5cc96082811f7ef620cf60/src/Common.sol#L595))

### Debugger

Key bindings are described in the `getting-started/` README.

![image](https://github.com/arr4n/specops/assets/519948/5057ad0f-bb6f-438b-a295-8b1f410d2330)

## Acknowledgements

Some of SpecOps was, of course, inspired by
[Huff](https://github.com/huff-language). I hope to provide something different,
of value, and to inspire them too.
