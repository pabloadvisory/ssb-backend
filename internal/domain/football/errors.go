package football

import "errors"

var (
	ErrNotFound = errors.New("football resource not found")
	ErrConflict = errors.New("football resource conflicts with current state")
	ErrInvalid  = errors.New("invalid football data")
)
