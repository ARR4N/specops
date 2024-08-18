package main

import (
	"github.com/ethereum/go-ethereum/common"

	. "github.com/arr4n/specops" //lint:ignore ST1001 SpecOps DSL is designed to be dot-imported
	"github.com/arr4n/specops/specopscli"
	"github.com/arr4n/specops/stack"
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
	specopscli.Run(code())
}

// Stop unused imports being removed.
var (
	_ = stack.ExpectDepth(0)
	_ = common.Address{}
)
