package specops

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"github.com/arr4n/specops/evmdebug"
	"github.com/arr4n/specops/revert"
	"github.com/arr4n/specops/runopts"
)

// Run calls c.Compile() and runs the compiled bytecode on a freshly
// instantiated [vm.EVM]. See [runopts] for configuring the EVM and call
// parameters, and for intercepting bytecode.
//
// Run returns an error if the code reverts. The error will be a [revert.Error]
// carrying the same revert error and data as the [core.ExecutionResult]
// returned by Run. To only return errors in the [core.ExecutionResult], use
// [runopts.NoErrorOnRevert].
func (c Code) Run(callData []byte, opts ...runopts.Option) (*core.ExecutionResult, error) {
	compiled, err := c.Compile()
	if err != nil {
		return nil, fmt.Errorf("%T.Compile(): %v", c, err)
	}
	return runBytecode(compiled, callData, opts...)
}

// StartDebugging appends a runopts.Debugger (`dbg`) to the Options, calls
// c.Run() in a new goroutine, and returns `dbg` along with a function to
// retrieve the results of Run(). The function will block until Run() returns,
// i.e. when dbg.Done() returns true. There is no need to call dbg.Wait().
//
// If execution never completes, such that dbg.Done() always returns false, then
// the goroutine will be leaked.
//
// Any compilation error will be returned by StartDebugging() while execution
// errors are returned by a call to the returned function. Said execution errors
// can be errors.Unwrap()d to access the same error available in
// `dbg.State().Err`.
func (c Code) StartDebugging(callData []byte, opts ...runopts.Option) (*evmdebug.Debugger, func() (*core.ExecutionResult, error), error) {
	compiled, err := c.Compile()
	if err != nil {
		return nil, nil, fmt.Errorf("%T.Compile(): %v", c, err)
	}

	dbg, opt := runopts.WithNewDebugger()
	opts = append(opts, opt)

	var (
		result *core.ExecutionResult
		resErr error
	)
	done := make(chan struct{})
	go func() {
		result, resErr = runBytecode(compiled, callData, opts...)
		close(done)
	}()

	dbg.Wait()

	return dbg, func() (*core.ExecutionResult, error) {
		<-done
		return result, resErr
	}, nil
}

// RunTerminalDebugger is equivalent to StartDebugging(), but instead of
// returning the Debugger and results function, it calls
// Debugger.RunTerminalUI().
func (c Code) RunTerminalDebugger(callData []byte, opts ...runopts.Option) error {
	bytecode := runopts.CaptureBytecode()
	opts = append(opts, bytecode)
	dbg, results, err := c.StartDebugging(callData, opts...)
	if err != nil {
		return err
	}
	defer dbg.FastForward()

	dbgCtx := &evmdebug.Context{
		CallData: callData,
		Bytecode: bytecode.Val,
		Results:  results,
	}
	return dbg.RunTerminalUI(dbgCtx)
}

func runBytecode(compiled, callData []byte, opts ...runopts.Option) (*core.ExecutionResult, error) {
	cfg, err := newRunConfig(compiled, opts...)
	if err != nil {
		return nil, err
	}
	evm := vm.NewEVM(
		cfg.BlockCtx,
		cfg.TxCtx,
		cfg.StateDB,
		cfg.ChainConfig,
		cfg.VMConfig,
	)

	gp := core.GasPool(30e6)
	msg := &core.Message{
		To:    &cfg.Contract.Address,
		From:  cfg.From,
		Value: cfg.Value.ToBig(),
		Data:  callData,
		// Not configurable but necessary
		GasFeeCap: big.NewInt(0),
		GasTipCap: big.NewInt(0),
		GasPrice:  big.NewInt(0),
		GasLimit:  gp.Gas(),
	}

	res, err := core.ApplyMessage(evm, msg, &gp)
	if err != nil {
		return nil, err
	}
	if cfg.NoErrorOnRevert {
		return res, nil
	}
	return res, revert.ErrFrom(res) /* may be nil */
}

func newRunConfig(compiled []byte, opts ...runopts.Option) (*runopts.Configuration, error) {
	db := state.NewDatabase(rawdb.NewMemoryDatabase())
	sdb, err := state.New(common.Hash{}, db, nil)
	if err != nil {
		return nil, err
	}

	cfg := &runopts.Configuration{
		StateDB:  sdb,
		Contract: runopts.NewContract(compiled),
		From:     runopts.DefaultFromAddress(),
		Value:    uint256.NewInt(0),
		BlockCtx: vm.BlockContext{
			BlockNumber: big.NewInt(0),
			Random:      &common.Hash{}, // required post merge
			BaseFee:     big.NewInt(0),
			CanTransfer: func(sdb vm.StateDB, a common.Address, val *uint256.Int) bool {
				return sdb.GetBalance(a).Cmp(val) != -1
			},
			Transfer: func(sdb vm.StateDB, from, to common.Address, val *uint256.Int) {
				sdb.SubBalance(from, val, tracing.BalanceChangeTransfer)
				sdb.AddBalance(to, val, tracing.BalanceChangeTransfer)
			},
		},
		ChainConfig: &params.ChainConfig{
			LondonBlock: big.NewInt(0),
			CancunTime:  new(uint64),
		},
	}
	for _, o := range opts {
		if err := o.Apply(cfg); err != nil {
			return nil, fmt.Errorf("runopts.Option[%T].Apply(): %v", o, err)
		}
	}

	a := cfg.Contract.Address
	if !sdb.Exist(a) {
		sdb.CreateAccount(a)
	}
	sdb.CreateContract(a)
	if len(sdb.GetCode(a)) > 0 {
		return nil, fmt.Errorf("runopts.Options MUST NOT set the code of the contract")
	}
	sdb.SetCode(a, compiled)
	sdb.AddAddressToAccessList(a)

	sdb.AddBalance(cfg.From, cfg.Value, tracing.BalanceChangeUnspecified)

	return cfg, nil
}
