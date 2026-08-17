package domain

import "time"

type Project struct {
	ID            string
	DisplayName   string
	RelativePath  string
	AbsolutePath  string
	WorkspaceRoot string
	LastSeenAt    time.Time
}

type Session struct {
	ID        string
	ProjectID string
	Title     string
	Directory string
}

type RuntimeState struct {
	WorkspaceRoot string
	ProjectID     string
	RelativePath  string
	SessionID     string
	UpdatedAt     time.Time
}

type DirectoryEntry struct {
	Name         string
	RelativePath string
}

type NavigationState struct {
	ID                  string
	ChatID              int64
	CurrentRelativePath string
	ExpiresAt           time.Time
	CreatedAt           time.Time
}

type AppConfig struct {
	WorkspaceRoot string
	StatePath     string
	TelegramToken string
	AllowedChatID int64
	OpenCodePort  int
}

type HealthStatus struct {
	Healthy bool
	Version string
}

type FileChange struct {
	Path   string
	Status string
}

type OpenCodeEvent struct {
	Type      string
	ProjectID string
	SessionID string
	Data      []byte
}

type BotButton struct {
	Text string
	Data string
}

type BotResponse struct {
	Text    string
	Buttons [][]BotButton
	Edit    bool
}
