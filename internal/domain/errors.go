package domain

import "errors"

var (
	ErrOutsideWorkspace       = errors.New("path is outside workspace")
	ErrNotDirectory           = errors.New("path is not a directory")
	ErrNavigationNotFound     = errors.New("navigation state not found")
	ErrUnauthorizedNavigation = errors.New("navigation state belongs to another chat")
	ErrServerNotRunning       = errors.New("opencode server is not running")
	ErrSessionRequired        = errors.New("no active session")
	ErrProjectRequired        = errors.New("no active project")
	ErrWorkspaceNotConfigured = errors.New("workspace not configured")
)
