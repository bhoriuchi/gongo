package gongo

import "errors"

// ErrInvalidTarget invalid target pointer
var ErrInvalidTarget = errors.New("target must be a non-nil pointer")

// ErrIncompleteData bad data
var ErrIncompleteData = errors.New("required data was incomplete")
