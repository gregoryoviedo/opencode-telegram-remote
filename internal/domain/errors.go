package domain

import "errors"

var (
	ErrOutsideWorkspace   = errors.New("path is outside workspace")
	ErrNotDirectory       = errors.New("path is not a directory")
	ErrNavigationNotFound = errors.New("navigation state not found")
)
