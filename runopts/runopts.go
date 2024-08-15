// Package runopts provides configuration options for specops.Code.Run().
package runopts

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"github.com/solidifylabs/specops/evmdebug"
)

// A Configuration carries all values that can be modified to configure a call
// to specops.Code.Run(). It is intially set by Run() and then passed to all
// Options to be modified.
type Configuration struct {
	Contract      *Contract
	From          common.Address
	Value         *uint256.Int
	ErrorOnRevert bool // see Run() re errors
	// vm.NewEVM()
	BlockCtx    vm.BlockContext
	TxCtx       vm.TxContext
	StateDB     vm.StateDB
	ChainConfig *params.ChainConfig
	VMConfig    vm.Config
}

// Contract defines how the compiled SpecOps bytecode will be "deployed" before
// being run.
type Contract struct {
	Address  common.Address
	bytecode []byte
}

// NewContract returns a new [Contract] with the specified bytecode.
func NewContract(bytecode []byte) *Contract {
	return &Contract{
		Address:  DefaultContractAddress(),
		bytecode: bytecode,
	}
}

// Bytecode returns a copy of the code to be deployed.
func (c *Contract) Bytecode() []byte {
	return common.CopyBytes(c.bytecode)
}

// DefaultContractAddress returns the default address used as
// [Contract.Address].
func DefaultContractAddress() common.Address {
	return addressFromString("specops:contract")
}

// DefaultFromAddress returns the default address from which the contract is
// called.
func DefaultFromAddress() common.Address {
	return addressFromString("specops:from")
}

func addressFromString(s string) common.Address {
	return common.BytesToAddress(crypto.Keccak256([]byte(s)))
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

// WithDebugger returns an Option that sets Configuration.VMConfig.Tracer to
// dbg.Tracer(), intercepting every opcode execution. See evmdebug for details.
func WithDebugger(dbg *evmdebug.Debugger) Option {
	return Func(func(c *Configuration) error {
		c.VMConfig.Tracer = dbg.Tracer()
		return nil
	})
}

// WithNewDebugger is a convenience function for constructing a new Debugger,
// passing it to WithDebugger(), and returning both the Debugger and the Option.
func WithNewDebugger() (*evmdebug.Debugger, Option) {
	d := evmdebug.NewDebugger()
	return d, WithDebugger(d)
}

func ErrorOnRevert() Option {
	return Func(func(c *Configuration) error {
		c.ErrorOnRevert = true
		return nil
	})
}

func ContractAddress(a common.Address) Option {
	return Func(func(c *Configuration) error {
		c.Contract.Address = a
		return nil
	})
}

func From(a common.Address) Option {
	return Func(func(c *Configuration) error {
		c.From = a
		return nil
	})
}

type Number interface {
	uint256.Int | *uint256.Int | uint | uint64
}

func Value[N Number](v N) Option {
	var u *uint256.Int
	switch v := any(v).(type) {
	case uint256.Int:
		u = &v
	case *uint256.Int:
		u = v
	case uint:
		u = uint256.NewInt(uint64(v))
	case uint64:
		u = uint256.NewInt(v)
	}

	return Func(func(c *Configuration) error {
		c.Value = u
		return nil
	})
}

func GenesisAlloc(alloc types.GenesisAlloc) Option {
	return Func(func(c *Configuration) error {
		s := c.StateDB
		for addr, acc := range alloc {
			s.CreateAccount(addr)
			if len(acc.Code) > 0 {
				s.SetCode(addr, acc.Code)
			}
			for slot, val := range acc.Storage {
				s.SetState(addr, slot, val)
			}
			if b := acc.Balance; b != nil {
				s.AddBalance(addr, uint256.MustFromBig(b), tracing.BalanceChangeUnspecified)
			}
			s.SetNonce(addr, acc.Nonce)
		}
		return nil
	})
}

type Captured[T any] struct {
	Val T

	apply Func
}

var _ Option = (*Captured[struct{}])(nil)

func (c *Captured[T]) Apply(cfg *Configuration) error {
	return c.apply(cfg)
}

func Capture[T any](fn func(*Configuration) T) *Captured[T] {
	c := new(Captured[T])
	c.apply = func(cfg *Configuration) error {
		c.Val = fn(cfg)
		return nil
	}
	return c
}

func CaptureConfig() *Captured[*Configuration] {
	return Capture(func(c *Configuration) *Configuration {
		return c
	})
}

func CaptureBytecode() *Captured[[]byte] {
	return Capture(func(c *Configuration) []byte {
		return c.Contract.Bytecode()
	})
}

func CaptureStateDB() *Captured[vm.StateDB] {
	return Capture(func(c *Configuration) vm.StateDB {
		return c.StateDB
	})
}
