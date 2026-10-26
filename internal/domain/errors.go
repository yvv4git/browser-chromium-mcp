package domain

import "errors"

var (
	ErrSessionNotFound      = errors.New("session not found")
	ErrTabNotFound          = errors.New("tab not found")
	ErrElementNotFound      = errors.New("element not found")
	ErrInvalidRef           = errors.New("invalid element ref")
	ErrEmptySelector        = errors.New("ref or selector is required")
	ErrUnsupportedKey       = errors.New("unsupported key")
	ErrUnsupportedDirection = errors.New("unsupported scroll direction")
	ErrUnsupportedFormat    = errors.New("unsupported image format")
)
