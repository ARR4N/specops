package revert

import "github.com/ethereum/go-ethereum/core"

type Error struct {
	Data []byte
	Err  error
}

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

func (e *Error) Error() string {
	return e.Err.Error()
}

func (e *Error) Unwrap() error {
	return e.Err
}
