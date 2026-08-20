package opencode_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/gregoryoviedo/opencode-telegram-remote/internal/adapter/opencode"
)

func newDiscardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func portFromURL(t *testing.T, rawURL string) int {
	t.Helper()
	idx := strings.LastIndex(rawURL, ":")
	if idx < 0 {
		t.Fatalf("cannot extract port from %q", rawURL)
	}
	port, err := strconv.Atoi(rawURL[idx+1:])
	if err != nil {
		t.Fatalf("parse port from %q: %v", rawURL, err)
	}
	return port
}

func TestManagerStartInWorkingDirAndStops(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/global/health") {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"healthy":true,"version":"x"}`)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	manager := opencode.NewManager(opencode.ManagerOptions{
		Bin:    "opencode",
		Port:   portFromURL(t, srv.URL),
		Logger: newDiscardLogger(),
	})
	dir := t.TempDir()

	if err := manager.Start(context.Background(), dir); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if !manager.StartedSubprocess() {
		t.Fatal("StartedSubprocess=false after Start")
	}
	if got := manager.WorkingDir(); got != dir {
		t.Fatalf("WorkingDir=%q, want %q", got, dir)
	}

	if err := manager.Start(context.Background(), dir); err != nil {
		t.Fatalf("Start (re-issue): %v", err)
	}
	if !manager.StartedSubprocess() {
		t.Fatal("StartedSubprocess=false after second Start")
	}

	manager.Stop()
	if manager.StartedSubprocess() {
		t.Fatal("StartedSubprocess=true after Stop")
	}
	if got := manager.WorkingDir(); got != "" {
		t.Fatalf("WorkingDir=%q, want empty", got)
	}
}

func TestManagerStartRejectsEmptyWorkingDir(t *testing.T) {
	manager := opencode.NewManager(opencode.ManagerOptions{
		Bin:    "opencode",
		Port:   4096,
		Logger: newDiscardLogger(),
	})
	if err := manager.Start(context.Background(), ""); err == nil {
		t.Fatal("expected error for empty working dir")
	}
}

func TestManagerStartRejectsNonDirectory(t *testing.T) {
	manager := opencode.NewManager(opencode.ManagerOptions{
		Bin:    "opencode",
		Port:   4096,
		Logger: newDiscardLogger(),
	})
	path := filepath.Join(t.TempDir(), "file.txt")
	if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := manager.Start(context.Background(), path); err == nil {
		t.Fatal("expected error for file-as-working-dir")
	}
}
