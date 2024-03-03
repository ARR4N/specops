// Package runopts provides configuration options for specops.Code.Run().
package runopts

import (
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
)

// A Configuration carries all values that can be modified to configure a call
// to specops.Code.Run(). It is intially set by Run() and then passed to all
// Options to be modified.
type Configuration struct {
	Contract *vm.Contract
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

// A Func converts a function into an Option by calling itself as Apply().
type Func func(*Configuration) error

// Apply returns f(c).
func (f Func) Apply(c *Configuration) error {
	return f(c)
}

// ReadOnly sets the `readOnly` argument to true when calling
// EVMInterpreter.Run(), equivalent to a static call.
func ReadOnly() Option {
	return Func(func(c *Configuration) error {
		c.ReadOnly = true
		return nil
	})
}
