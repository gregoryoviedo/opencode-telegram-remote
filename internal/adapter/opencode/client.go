package opencode

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/opencode-remote/opencode-telegram-remote/internal/domain"
)

const (
	defaultTimeout         = 15 * time.Second
	defaultReconnectWait   = time.Second
	defaultPromptRequestTimeout = 10 * time.Minute
)

type Client struct {
	baseURL string
	http    *http.Client
	stream  *http.Client
	prompt  *http.Client
}

type promptPayload struct {
	Parts []promptPart `json:"parts"`
}

type promptPart struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type assistantMessageDTO struct {
	Info  messageInfo `json:"info"`
	Parts []partDTO   `json:"parts"`
}

type partDTO struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type createSessionPayload struct {
	ParentID string `json:"parentID,omitempty"`
}

type revertPayload struct {
	MessageID string `json:"messageID"`
}

// API response shapes. Only the fields we use are extracted.

type projectDTO struct {
	ID        string `json:"id"`
	Worktree  string `json:"worktree"`
	Time      struct {
		Updated int64 `json:"updated"`
	} `json:"time"`
}

type sessionSummary struct {
	Additions int `json:"additions"`
	Deletions int `json:"deletions"`
	Files     int `json:"files"`
}

type sessionDTO struct {
	ID        string         `json:"id"`
	ProjectID string         `json:"projectID"`
	Title     string         `json:"title"`
	Directory string         `json:"directory"`
	Slug      string         `json:"slug"`
	Summary   sessionSummary `json:"summary"`
	Time      struct {
		Created int64 `json:"created"`
		Updated int64 `json:"updated"`
	} `json:"time"`
}

type messageInfo struct {
	ID        string `json:"id"`
	SessionID string `json:"sessionID"`
	Role      string `json:"role"`
}

type messageDTO struct {
	Info messageInfo `json:"info"`
}

type fileChangeDTO struct {
	Path   string `json:"path"`
	Added  string `json:"added,omitempty"`
	Removed string `json:"removed,omitempty"`
	Status string `json:"status,omitempty"`
}

func NewClient(baseURL string, httpClient *http.Client) (*Client, error) {
	parsed, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("invalid OpenCode base URL: %q", baseURL)
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultTimeout}
	}
	streamClient := &http.Client{Transport: httpClient.Transport}
	promptClient := &http.Client{Transport: httpClient.Transport} // no hard cap; controlled via context
	return &Client{
		baseURL: parsed.String(),
		http:    httpClient,
		stream:  streamClient,
		prompt:  promptClient,
	}, nil
}

func (c *Client) Health(ctx context.Context) (domain.HealthStatus, error) {
	var response struct {
		Healthy bool   `json:"healthy"`
		Version string `json:"version"`
	}
	if err := c.getJSON(ctx, "/global/health", &response); err != nil {
		return domain.HealthStatus{}, err
	}
	return domain.HealthStatus{Healthy: response.Healthy, Version: response.Version}, nil
}

func (c *Client) ListProjects(ctx context.Context) ([]domain.Project, error) {
	var dto []projectDTO
	if err := c.getJSON(ctx, "/project", &dto); err != nil {
		return nil, err
	}
	projects := make([]domain.Project, 0, len(dto))
	for _, p := range dto {
		updated := time.UnixMilli(p.Time.Updated).UTC()
		projects = append(projects, domain.Project{
			ID:           p.ID,
			DisplayName:  filepathBaseName(p.Worktree),
			AbsolutePath: p.Worktree,
			LastSeenAt:   updated,
		})
	}
	return projects, nil
}

func (c *Client) ListSessions(ctx context.Context) ([]domain.Session, error) {
	var dto []sessionDTO
	if err := c.getJSON(ctx, "/session", &dto); err != nil {
		return nil, err
	}
	sessions := make([]domain.Session, 0, len(dto))
	for _, s := range dto {
		sessions = append(sessions, sessionFromDTO(s))
	}
	return sessions, nil
}

func (c *Client) CreateSession(ctx context.Context, parentID string) (domain.Session, error) {
	var dto sessionDTO
	if err := c.doJSON(ctx, http.MethodPost, "/session", createSessionPayload{ParentID: parentID}, &dto); err != nil {
		return domain.Session{}, err
	}
	return sessionFromDTO(dto), nil
}

func (c *Client) SendPrompt(ctx context.Context, sessionID, text string) (string, error) {
	promptCtx, cancel := context.WithTimeout(ctx, defaultPromptRequestTimeout)
	defer cancel()

	payload := promptPayload{Parts: []promptPart{{Type: "text", Text: text}}}
	path := "/session/" + url.PathEscape(sessionID) + "/message"

	var response assistantMessageDTO
	if err := c.doJSONWith(c.prompt, promptCtx, http.MethodPost, path, payload, &response); err != nil {
		return "", err
	}
	return joinAssistantText(response.Parts), nil
}

func joinAssistantText(parts []partDTO) string {
	var b strings.Builder
	for _, part := range parts {
		if part.Type != "text" || part.Text == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(part.Text)
	}
	return b.String()
}

func (c *Client) Revert(ctx context.Context, sessionID string) error {
	path := "/session/" + url.PathEscape(sessionID) + "/message"
	var messages []messageDTO
	if err := c.getJSON(ctx, path, &messages); err != nil {
		return fmt.Errorf("list messages for revert: %w", err)
	}
	var lastUser string
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Info.Role == "user" {
			lastUser = messages[i].Info.ID
			break
		}
	}
	if lastUser == "" {
		return errors.New("no user message found in session; nothing to revert")
	}
	revertPath := "/session/" + url.PathEscape(sessionID) + "/revert"
	return c.doJSON(ctx, http.MethodPost, revertPath, revertPayload{MessageID: lastUser}, nil)
}

func (c *Client) FileStatus(ctx context.Context, sessionID string) ([]domain.FileChange, error) {
	var dto []fileChangeDTO
	path := "/session/" + url.PathEscape(sessionID) + "/diff"
	if err := c.getJSON(ctx, path, &dto); err != nil {
		return nil, err
	}
	changes := make([]domain.FileChange, 0, len(dto))
	for _, change := range dto {
		status := change.Status
		if status == "" {
			switch {
			case change.Removed != "":
				status = "deleted"
			case change.Added != "":
				status = "added"
			default:
				status = "modified"
			}
		}
		changes = append(changes, domain.FileChange{Path: change.Path, Status: status})
	}
	return changes, nil
}

func (c *Client) SubscribeEvents(ctx context.Context) (<-chan domain.OpenCodeEvent, error) {
	if _, err := url.ParseRequestURI(c.baseURL); err != nil {
		return nil, err
	}
	output := make(chan domain.OpenCodeEvent)
	go c.streamEvents(ctx, output)
	return output, nil
}

func (c *Client) streamEvents(ctx context.Context, output chan<- domain.OpenCodeEvent) {
	defer close(output)
	for {
		if ctx.Err() != nil {
			return
		}
		if err := c.readEventStream(ctx, output); err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(defaultReconnectWait):
		}
	}
}

func (c *Client) readEventStream(ctx context.Context, output chan<- domain.OpenCodeEvent) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/global/event", nil)
	if err != nil {
		return err
	}
	request.Header.Set("Accept", "text/event-stream")
	response, err := c.stream.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return fmt.Errorf("OpenCode events returned %s: %s", response.Status, strings.TrimSpace(string(body)))
	}

	scanner := bufio.NewScanner(response.Body)
	scanner.Buffer(make([]byte, 4096), 1024*1024)
	var eventType string
	var data bytes.Buffer
	flush := func() error {
		if data.Len() == 0 {
			eventType = ""
			return nil
		}
		event := domain.OpenCodeEvent{Type: eventType, Data: append([]byte(nil), data.Bytes()...)}
		select {
		case output <- event:
		case <-ctx.Done():
			return ctx.Err()
		}
		eventType = ""
		data.Reset()
		return nil
	}
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if err := flush(); err != nil {
				return err
			}
			continue
		}
		if strings.HasPrefix(line, ":") {
			continue
		}
		switch {
		case strings.HasPrefix(line, "event:"):
			eventType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		case strings.HasPrefix(line, "data:"):
			if data.Len() > 0 {
				data.WriteByte('\n')
			}
			data.WriteString(strings.TrimPrefix(line, "data:"))
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return flush()
}

func (c *Client) getJSON(ctx context.Context, path string, target any) error {
	return c.doJSONWith(c.http, ctx, http.MethodGet, path, nil, target)
}

func (c *Client) doJSON(ctx context.Context, method, path string, payload, target any) error {
	return c.doJSONWith(c.http, ctx, method, path, payload, target)
}

func (c *Client) doJSONWith(client *http.Client, ctx context.Context, method, path string, payload, target any) error {
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("encode OpenCode request: %w", err)
		}
		body = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return err
	}
	request.Header.Set("Accept", "application/json")
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("request OpenCode %s %s: %w", method, path, err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return fmt.Errorf("OpenCode %s returned %s: %s", path, response.Status, strings.TrimSpace(string(body)))
	}
	if target == nil || response.StatusCode == http.StatusNoContent {
		return nil
	}
	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		return fmt.Errorf("decode OpenCode %s response: %w", path, err)
	}
	return nil
}

func sessionFromDTO(dto sessionDTO) domain.Session {
	return domain.Session{
		ID:        dto.ID,
		ProjectID: dto.ProjectID,
		Title:     dto.Title,
		Directory: dto.Directory,
	}
}

func filepathBaseName(path string) string {
	if idx := strings.LastIndex(path, "/"); idx >= 0 {
		return path[idx+1:]
	}
	return path
}
