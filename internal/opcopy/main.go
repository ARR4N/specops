// The opcopy binary generates a Go file for use in the `specops` package.
// It mirrors all EVM opcodes that don't have special representations, and
// provides a mapping from all opcodes to the number of values they pop/push
// from the stack.
package main

import (
	"fmt"
	"os"
	"text/template"

	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	type opParams struct {
		Op        vm.OpCode
		Pop, Push uint
		Special   bool
	}
	var ops []*opParams

	for i := 0; i < 256; i++ {
		o := vm.OpCode(i)
		if vm.StringToOp(o.String()) != o { // invalid opcode
			continue
		}
		ops = append(ops, &opParams{
			Op:      o,
			Special: (o.IsPush() && o != vm.PUSH0) || o == vm.JUMPDEST,
		})
	}

	rules := params.Rules{IsCancun: true}
	jumpTable, err := vm.LookupInstructionSet(rules)
	if err != nil {
		return fmt.Errorf("go-ethereum/core/vm.LookupInstructionSet(%+v): %v", rules, err)
	}
	for _, o := range ops {
		minStack, maxStack := jumpTable[o.Op].Stack()

		switch o.Op & 0xf0 {
		case vm.DUP1:
			// See comment in generated code.
			o.Pop = 1
			o.Push = 2
		case vm.SWAP1:
			o.Pop = 1
			o.Push = 1
		default:
			// Invert the derivation of minStack/maxStack from pop/push:
			// https://github.com/ethereum/go-ethereum/blob/57d2b552c74dbd03b9909e6b8cd7b3de1f8b40e9/core/vm/stack_table.go
			o.Pop = uint(minStack)
			o.Push = uint(params.StackLimit) + o.Pop - uint(maxStack)
		}
	}

	tmpl := template.Must(template.New("go").Parse(`package specops

//
// GENERATED CODE - DO NOT EDIT
//

import (
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/arr4n/specops/types"
)

// Aliases of all regular vm.OpCode constants that don't have "special" replacements.
const (
{{- range .}}{{if not .Special}}
	{{.Op.String}} = types.OpCode(vm.{{.Op.String}})
{{- end}}{{end}}
)

// stackDeltas maps all valid vm.OpCode values to the number of values they
// pop and then push from/to the stack.
//
// Although DUPs technically only push a single value and SWAPs none, they are
// recorded as popping and pushing one more than they actually do as this
// implies a minimum stack depth to begin with but with the same effective
// change.
var stackDeltas = map[vm.OpCode]stackDelta{
{{- range .}}
	vm.{{.Op.String}}: {pop: {{.Pop}}, push: {{.Push}}},
{{- end}}
}
`))

	if err := tmpl.Execute(os.Stdout, ops); err != nil {
		return fmt.Errorf("%T.Execute(os.Stdout, …): %v", tmpl, err)
	}
	return nil
}
