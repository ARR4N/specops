# SpecialOps [![Go](https://github.com/solidifylabs/specialops/actions/workflows/go.yml/badge.svg)](https://github.com/solidifylabs/specialops/actions/workflows/go.yml) [![Go Reference](https://pkg.go.dev/badge/github.com/solidifylabs/specialops.svg)](https://pkg.go.dev/github.com/solidifylabs/specialops)

> `specialops` is a domain-specific language for crafting EVM bytecode in Go.

This is a _very_ early release. In fact, it's just a weekend project gone rogue
so is less than a week old.

## _special_ opcodes

Writing bytecode is hard. There's always that temptation to give up and use a
higher-level language with all of its conveniences, but that defeats the point.
What if we could maintain full control of the opcode placement, but with
syntactic sugar to help the medicine go down?

*Special* opcodes provide just that. Some are interpreted by the compiler,
converting them to
[regular](https://pkg.go.dev/github.com/ethereum/go-ethereum/core/vm#OpCode)
equivalents, while others are simply compiler hints that leave the resulting
bytecode unchanged.

## Features

- [x] `JUMPDEST` labels (absolute)
- [ ] `JUMPDEST` labels (relative to `PC`)
- [x] Function-like syntax (optional)
- [x] Inverted `DUP`/`SWAP` special opcodes from "bottom" of stack (i.e. pseudo-variables)
- [x] `PUSH<T>` for native Go types
- [x] Macros
- [x] Compiler-state assertions (e.g. expected stack depth)
- [ ] Automatic stack permutation
- [ ] Standalone compiler

## Hello world

The `specialops` Go package has a minimal footprint to allow for dot-importing,
making all exported symbols available. TODO: expand on the implications,
rationale, and recommendations as this goes against the style guide.

```go
import . github.com/solidifylabs/specialops

…

hello := []byte("Hello world")
code := Code{
    // The compiler determines the shortest-possible PUSH<n> opcode.
    // Fn() simply reverses its arguments (a surprisingly powerful construct)!
    Fn(MSTORE, PUSH0, PUSH(hello)),
    Fn(RETURN, PUSH(32-len(hello)), PUSH(len(hello))),
}

byteCode, err := code.Compile()
```

TODO: link to `examples_test.go` or `pkg.go.dev` examples.

## Acknowledgements

Some of SpecialOps was, of course, inspired by
[Huff](https://github.com/huff-language). I hope to provide something different,
of value, and to inspire them too.