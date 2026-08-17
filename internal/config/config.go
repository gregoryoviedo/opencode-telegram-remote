package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	WorkspaceRoot   string
	StatePath       string
	TelegramToken   string
	AllowedChatID   int64
	OpenCodePort    int
	OpenCodeBin     string
	AutoStart       bool
	EnvFile         string
}

func Load() (*Config, error) {
	config := &Config{
		OpenCodePort: 4096,
		OpenCodeBin:  "opencode",
		AutoStart:    false,
	}

	if path, ok := locateEnv(); ok {
		if err := godotenv.Overload(path); err != nil {
			return nil, fmt.Errorf("load env file %q: %w", path, err)
		}
		config.EnvFile = path
	}

	config.WorkspaceRoot = os.Getenv("WORKSPACE_ROOT")
	if config.WorkspaceRoot == "" {
		return nil, errors.New("WORKSPACE_ROOT is required (set it in .env or environment)")
	}
	config.WorkspaceRoot = filepath.Clean(config.WorkspaceRoot)

	config.StatePath = os.Getenv("REMOTE_STATE_PATH")
	if config.StatePath == "" {
		config.StatePath = filepath.Join(config.WorkspaceRoot, ".opencode-remote", "state.db")
	}

	config.TelegramToken = os.Getenv("TELEGRAM_BOT_TOKEN")
	if config.TelegramToken == "" {
		return nil, errors.New("TELEGRAM_BOT_TOKEN is required")
	}

	chatIDRaw := os.Getenv("ALLOWED_CHAT_ID")
	if chatIDRaw == "" {
		return nil, errors.New("ALLOWED_CHAT_ID is required")
	}
	chatID, err := strconv.ParseInt(chatIDRaw, 10, 64)
	if err != nil || chatID == 0 {
		return nil, fmt.Errorf("ALLOWED_CHAT_ID must be a non-zero integer: %q", chatIDRaw)
	}
	config.AllowedChatID = chatID

	if raw := os.Getenv("OPENCODE_PORT"); raw != "" {
		port, err := strconv.Atoi(raw)
		if err != nil || port < 1 || port > 65535 {
			return nil, fmt.Errorf("OPENCODE_PORT must be between 1 and 65535: %q", raw)
		}
		config.OpenCodePort = port
	}

	if raw := os.Getenv("OPENCODE_BIN"); raw != "" {
		config.OpenCodeBin = raw
	}

	if raw := os.Getenv("OPENCODE_AUTOSTART"); raw != "" {
		parsed, err := parseBool(raw)
		if err != nil {
			return nil, fmt.Errorf("OPENCODE_AUTOSTART must be a boolean: %q", raw)
		}
		config.AutoStart = parsed
	}

	return config, nil
}

func parseBool(raw string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "on":
		return true, nil
	case "0", "false", "no", "off":
		return false, nil
	}
	return false, errors.New("invalid boolean")
}

func locateEnv() (string, bool) {
	if path := os.Getenv("ENV_FILE"); path != "" {
		if _, err := os.Stat(path); err == nil {
			return path, true
		}
	}
	if _, err := os.Stat(".env"); err == nil {
		abs, _ := filepath.Abs(".env")
		return abs, true
	}
	return "", false
}
