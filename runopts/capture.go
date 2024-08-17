package runopts

import "github.com/ethereum/go-ethereum/core/vm"

// A Captured value is an [Option] that stores part of the [Configuration] for
// later inspection. After Run() and similar functions return, the Val field
// will be populated.
//
// A set of constructors is provided for commonly captured values.
type Captured[T any] struct {
	Val T

	apply Func
}

var _ Option = (*Captured[struct{}])(nil)

// Apply implements the [Option] interface, storing the value to be captured.
func (c *Captured[T]) Apply(cfg *Configuration) error {
	return c.apply(cfg)
}

// Capture returns a Captured value that is valid _after_ being passed as an
// option to Run(). [fn] must extract and return the value to capture.
func Capture[T any](fn func(*Configuration) T) *Captured[T] {
	c := new(Captured[T])
	c.apply = func(cfg *Configuration) error {
		c.Val = fn(cfg)
		return nil
	}
	return c
}

// CaptureConfig captures the entire [Configuration].
func CaptureConfig() *Captured[*Configuration] {
	return Capture(func(c *Configuration) *Configuration {
		return c
	})
}

// CaptureBytecode captures a copy of the compiled bytecode.
func CaptureBytecode() *Captured[[]byte] {
	return Capture(func(c *Configuration) []byte {
		return c.Contract.Bytecode()
	})
}

// CaptureStateDB captures the [vm.StateDB] used for storage of accounts (i.e.
// balances, code, storage, etc).
func CaptureStateDB() *Captured[vm.StateDB] {
	return Capture(func(c *Configuration) vm.StateDB {
		return c.StateDB
	})
}
