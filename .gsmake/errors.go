package tasks

import "errors"

// errors
var (
	ErrModuleNotFound = errors.New("gsrpc module not found")
	ErrUnknownPath    = errors.New("unknown path")
)
