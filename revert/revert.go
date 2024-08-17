// Package revert provides errors and error handling for EVM smart contracts
// that revert.
package revert

import (
	"errors"

	"github.com/ethereum/go-ethereum/core"
)

// An Error is an error signalling that code reverted.
type Error struct {
	Data []byte // [core.ExecutionResult.Revert()]
	Err  error  // [core.ExecutionResult.Err]
}

// Data returns the revert data from the error if it is an [Error]. The returned
// boolean indicates whether the possibly zero-length data was found; similar to
// the second return value from a map.
func Data(err error) (_ []byte, ok bool) {
	e := new(Error)
	if !errors.As(err, &e) {
		return nil, false
	}
	return e.Data, true
}

// ErrFrom converts a [core.ExecutionResult] into an error, or nil if the
// execution completely successfully. The returned error is non-nil i.f.f.
// r.Failed() is true.
func ErrFrom(r *core.ExecutionResult) error {
	if !r.Failed() {
		return nil
	}
	return &Error{
		Data: r.Revert(),
		Err:  r.Err,
	}
}

var _ error = (*Error)(nil)

// Error returns the error string from the [core.ExecutionResult.Err].
func (e *Error) Error() string {
	return e.Err.Error()
}

// Unwrap returns the wrapped [core.ExecutionResult.Err] value.
func (e *Error) Unwrap() error {
	return e.Err
}
