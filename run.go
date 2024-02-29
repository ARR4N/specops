package specialops

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
)

// Run calls c.Compile() and runs the compiled bytecode on a freshly
// instantiated vm.EVMInterpreter. The default EVM parameters MUST NOT be
// considered stable: they are currently such that code runs on the Cancun fork
// with no state DB.
func (c Code) Run(callData []byte) ([]byte, error) {
	compiled, err := c.Compile()
	if err != nil {
		return nil, fmt.Errorf("%T.Compile(): %v", c, err)
	}
	return runBytecode(compiled, callData, false)
}

func runBytecode(compiled, callData []byte, readOnly bool) ([]byte, error) {
	interp := vm.NewEVM(
		vm.BlockContext{
			BlockNumber: big.NewInt(1),
			Random:      &common.Hash{}, // non-nil -> post merge
		},
		vm.TxContext{},
		nil, /*statedb*/
		&params.ChainConfig{
			LondonBlock: big.NewInt(0),
			CancunTime:  new(uint64),
		},
		vm.Config{},
	).Interpreter()

	cc := &vm.Contract{
		Code: compiled,
		Gas:  30e6,
	}

	out, err := interp.Run(cc, callData, readOnly)
	if err != nil {
		return nil, fmt.Errorf("%T.Run([%T.Compile()], [callData], readOnly=%t): %v", interp, Code{}, readOnly, err)
	}
	return out, nil
}
