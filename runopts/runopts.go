// Package runopts provides configuration options for specialops.Code.Run().
package runopts

import (
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
)

// A Configuration carries all values that can be modified to configure a call
// to specialops.Code.Run(). It is intially set by Run() and then passed to all
// Options to be modified.
type Configuration struct {
	// vm.NewEVM()
	BlockCtx    vm.BlockContext
	TxCtx       vm.TxContext
	StateDB     vm.StateDB
	ChainConfig *params.ChainConfig
	VMConfig    vm.Config
	// EVMInterpreter.Run()
	ReadOnly bool // static call
}

// An Option modifies a Configuration.
type Option interface {
	Apply(*Configuration) error
}

// A FuncOption converts any function into an Option by calling itself as
// Apply().
type FuncOption func(*Configuration) error

// Apply returns f(c).
func (f FuncOption) Apply(c *Configuration) error {
	return f(c)
}

// ReadOnly sets the `readOnly` argument to true when calling
// EVMInterpreter.Run(), equivalent to a static call.
func ReadOnly() Option {
	return FuncOption(func(c *Configuration) error {
		c.ReadOnly = true
		return nil
	})
}
