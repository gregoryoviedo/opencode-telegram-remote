package domain

import (
	"context"
	"os"
)

type WorkspaceFS interface {
	ReadDir(name string) ([]os.DirEntry, error)
	EvalSymlinks(path string) (string, error)
	Stat(name string) (os.FileInfo, error)
}

type StateRepository interface {
	LoadRuntimeState(ctx context.Context) (RuntimeState, error)
	SaveRuntimeState(ctx context.Context, state RuntimeState) error
}

type NavigationRepository interface {
	SaveNavigation(ctx context.Context, state NavigationState) error
	GetNavigation(ctx context.Context, id string) (NavigationState, error)
	DeleteNavigation(ctx context.Context, id string) error
}

type OpenCodeClient interface {
	Health(ctx context.Context) (HealthStatus, error)
	ListProjects(ctx context.Context) ([]Project, error)
	ListSessions(ctx context.Context) ([]Session, error)
	CreateSession(ctx context.Context, parentID string) (Session, error)
	SendPrompt(ctx context.Context, sessionID, text string) (string, error)
	Revert(ctx context.Context, sessionID string) error
	FileStatus(ctx context.Context, sessionID string) ([]FileChange, error)
}

type BotHandler interface {
	HandleCommand(ctx context.Context, chatID int64, command string, args []string) (BotResponse, error)
	HandleText(ctx context.Context, chatID int64, text string) (BotResponse, error)
	HandleCallback(ctx context.Context, chatID int64, data string) (BotResponse, error)
}

// OpenCodeServerManager owns the lifecycle of the local OpenCode serve
// subprocess. Implementations are expected to swap the running subprocess
// atomically when Start is called again with a different working dir.
type OpenCodeServerManager interface {
	Start(ctx context.Context, workingDir string) error
	Stop()
	StartedSubprocess() bool
	WorkingDir() string
}
