package main

import (
	"github.com/ethereum/go-ethereum/common"

	. "github.com/solidifylabs/specops" //lint:ignore ST1001 SpecOps DSL is designed to be dot-imported
	"github.com/solidifylabs/specops/specopsdev"
	"github.com/solidifylabs/specops/stack"
)

func code() Code {
	// ----------------------------------------
	// ========== START EDITING HERE ==========
	// vvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvv

	hello := []byte("Hello, world!")

	return Code{
		Fn(MSTORE, PUSH0, PUSH(hello)),
		Fn(RETURN, PUSH(32-len(hello)), PUSH(len(hello))),
	}

	// ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
	// ========== STOP EDITING HERE ===========
	// ----------------------------------------
}

func main() {
	specopsdev.RunCLI(code())
}

// Stop unused imports being removed.
var (
	_ = stack.ExpectDepth(0)
	_ = common.Address{}
)
