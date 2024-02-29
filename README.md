# SpecialOps

> `specialops` is a domain-specific language for crafting EVM bytecode in Go.

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