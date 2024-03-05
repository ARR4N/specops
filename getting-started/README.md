# Getting started with SpecOps

1. [Install Go](https://go.dev/doc/install)
2. From the `getting-started` directory:

```shell
go run getting-started.spec.go compile
```

This will print the compiled EVM bytecode to `stdout`.

## Development

The `getting-started.spec.go` file contains everything you need to be productive.
If this is your first time using Go, stick between the `START/STOP EDITING HERE` comments and everything will work.

### Why do I have to learn Go?

> TL;DR You don't

SpecOps is a DSL built and run in Go, but designed so that it reads and is written like a standalone language.
The advantage of piggybacking on the Go toolchain is that we get all of the developer tooling out of the box: syntax highlighting, code completion, etc.
For more experienced Go developers, there is also support for native testing, interoperability with geth, etc.

A standalone language inside another?
In Go, all functions, types, etc. from external packages are *usually* referenced by their package name.
There is, however, the ability to "dot-import" a package, promoting these symbols such that the package-qualification is unnecessary.
`specops.Fn` becomes `Fn`, `specops.MSTORE` becomes `MSTORE`, etc. While this goes against the Go style guide, for a DSL it makes sense as it greatly improves developer experience.

## Other CLI usage

### Commands

The CLI has `compile`, `exec`, and `debug` commands. The `-h` or `--help` flag
will provide more information about each (for now, quite limited).

### calldata

Both the `exec` and `debug` commands support the `--calldata` flag, which accepts hex-encoded calldata. For example:

```shell
go run getting-started.spec.go debug --calldata decafc0ffeebad
```

### Debugging

* `<space>` Step to next instruction
* `<end>` Fast-forward to the end of execution
* `<Esc>` or `q` Once execution has ended, quit
* `Ctrl+C` At any time, quit