package news

import "errors"

var (
	ErrNotFound = errors.New("news article not found")
	ErrConflict = errors.New("news article conflicts with current state")
	ErrInvalid  = errors.New("invalid news article")
)
