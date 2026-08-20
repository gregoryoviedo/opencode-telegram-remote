package usecase

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gregoryoviedo/opencode-telegram-remote/internal/domain"
)

type Handler struct {
	navigation    *NavigationService
	state         domain.StateRepository
	opencode      domain.OpenCodeClient
	server        domain.OpenCodeServerManager
	browser       *WorkspaceBrowser
	workspaceRoot string
}

func NewHandler(navigation *NavigationService, state domain.StateRepository, opencode domain.OpenCodeClient, server domain.OpenCodeServerManager, browser *WorkspaceBrowser) *Handler {
	return &Handler{
		navigation:    navigation,
		state:         state,
		opencode:      opencode,
		server:        server,
		browser:       browser,
		workspaceRoot: browser.Root(),
	}
}

// StateForTest exposes the underlying StateRepository to tests in the same
// package. Production code must depend on the Handler port instead.
func (h *Handler) StateForTest() domain.StateRepository { return h.state }

func (h *Handler) HandleCommand(ctx context.Context, chatID int64, command string, args []string) (domain.BotResponse, error) {
	switch command {
	case "/start", "/help":
		return helpResponse(), nil
	case "/projects":
		state, entries, err := h.navigation.Start(ctx, chatID)
		if err != nil {
			return domain.BotResponse{}, err
		}
		return directoryResponse(state, entries), nil
	case "/init":
		return h.init(ctx, args)
	case "/status":
		return h.status(ctx)
	case "/sessions":
		return h.sessions(ctx, chatID, args)
	case "/diff", "/changes":
		return h.diff(ctx)
	case "/undo":
		return h.undo(ctx)
	default:
		return domain.BotResponse{Text: "Comando no reconocido. Usa /help."}, nil
	}
}

func helpResponse() domain.BotResponse {
	return domain.BotResponse{Text: strings.Join([]string{
		"OpenCode Remote listo.",
		"",
		"• /projects — selecciona la carpeta del proyecto.",
		"• /init — (re)arranca el servidor OpenCode en la carpeta activa.",
		"• /sessions — lista o crea sesiones del proyecto activo.",
		"• /status — salud del servidor y proyecto/sesión activos.",
		"• /diff — archivos modificados por la sesión activa.",
		"• /undo — revierte el último cambio.",
		"• texto libre — prompt directo a la sesión activa.",
	}, "\n")}
}

const telegramMaxMessageLen = 4096

func (h *Handler) HandleText(ctx context.Context, _ int64, text string) (domain.BotResponse, error) {
	if !h.server.StartedSubprocess() {
		return domain.BotResponse{Text: "El servidor OpenCode está apagado. Ejecuta /init."}, nil
	}
	state, err := h.state.LoadRuntimeState(ctx)
	if err != nil {
		return domain.BotResponse{}, err
	}
	if state.SessionID == "" {
		return domain.BotResponse{Text: "Selecciona primero una sesión con /sessions."}, nil
	}
	reply, err := h.opencode.SendPrompt(ctx, state.SessionID, text)
	if err != nil {
		return domain.BotResponse{Text: "OpenCode no pudo responder: " + err.Error()}, nil
	}
	if reply == "" {
		return domain.BotResponse{Text: "OpenCode terminó la respuesta sin texto (revisa /diff por si hubo cambios silenciosos)."}, nil
	}
	return domain.BotResponse{Text: truncateForTelegram(reply)}, nil
}

func truncateForTelegram(text string) string {
	if len(text) <= telegramMaxMessageLen {
		return text
	}
	return text[:telegramMaxMessageLen] + "\n\n... (truncado, sigue en OpenCode)"
}

func (h *Handler) HandleCallback(ctx context.Context, chatID int64, data string) (domain.BotResponse, error) {
	parts := strings.SplitN(data, "|", 3)
	if len(parts) < 2 {
		return expiredNavigation(), nil
	}
	switch parts[0] {
	case "e":
		if len(parts) != 3 {
			return expiredNavigation(), nil
		}
		state, entries, err := h.navigation.Enter(ctx, parts[1], chatID, parts[2])
		if err != nil {
			return navigationError(err), nil
		}
		return directoryResponse(state, entries), nil
	case "b":
		state, entries, err := h.navigation.Back(ctx, parts[1], chatID)
		if err != nil {
			return navigationError(err), nil
		}
		return directoryResponse(state, entries), nil
	case "h":
		state, entries, err := h.navigation.Home(ctx, parts[1], chatID)
		if err != nil {
			return navigationError(err), nil
		}
		return directoryResponse(state, entries), nil
	case "s":
		if len(parts) != 3 {
			return expiredNavigation(), nil
		}
		project, err := h.navigation.Select(ctx, parts[1], chatID, parts[2])
		if err != nil {
			return navigationError(err), nil
		}
		state, err := h.state.LoadRuntimeState(ctx)
		if err != nil {
			return domain.BotResponse{}, err
		}
		state.ProjectID = project.ID
		state.RelativePath = project.RelativePath
		state.WorkspaceRoot = project.WorkspaceRoot
		state.SessionID = ""
		if err := h.state.SaveRuntimeState(ctx, state); err != nil {
			return domain.BotResponse{}, err
		}
		return h.bindServer(ctx, project.AbsolutePath)
	case "sn":
		if len(parts) != 3 {
			return expiredNavigation(), nil
		}
		parsed, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil || parsed != chatID {
			return expiredNavigation(), nil
		}
		if !h.server.StartedSubprocess() {
			return domain.BotResponse{Text: "El servidor OpenCode está apagado. Ejecuta /init primero.", Edit: true}, nil
		}
		state, err := h.state.LoadRuntimeState(ctx)
		if err != nil {
			return domain.BotResponse{}, err
		}
		if state.ProjectID == "" {
			return domain.BotResponse{Text: "Selecciona primero un proyecto con /projects.", Edit: true}, nil
		}
		if parts[2] == "new" {
			session, err := h.opencode.CreateSession(ctx, "")
			if err != nil {
				return domain.BotResponse{}, err
			}
			state.SessionID = session.ID
			if err := h.state.SaveRuntimeState(ctx, state); err != nil {
				return domain.BotResponse{}, err
			}
			return domain.BotResponse{Text: "Nueva sesión activa: " + session.ID, Edit: true}, nil
		}
		state.SessionID = parts[2]
		if err := h.state.SaveRuntimeState(ctx, state); err != nil {
			return domain.BotResponse{}, err
		}
		return domain.BotResponse{Text: "Sesión activa: " + parts[2], Edit: true}, nil
	default:
		return expiredNavigation(), nil
	}
}

func directoryResponse(state domain.NavigationState, entries []domain.DirectoryEntry) domain.BotResponse {
	path := state.CurrentRelativePath
	if path == "" {
		path = "/"
	}
	response := domain.BotResponse{Text: fmt.Sprintf("Workspace: `%s`\n\nSelecciona una carpeta:", path)}
	for _, entry := range entries {
		response.Buttons = append(response.Buttons, []domain.BotButton{{Text: "📂 " + entry.Name, Data: "e|" + state.ID + "|" + entry.RelativePath}})
	}
	if state.CurrentRelativePath != "" {
		response.Buttons = append(response.Buttons, []domain.BotButton{{Text: "✅ Usar esta carpeta", Data: "s|" + state.ID + "|" + state.CurrentRelativePath}})
		response.Buttons = append(response.Buttons, []domain.BotButton{{Text: "⬅️ Atrás", Data: "b|" + state.ID}, {Text: "🏠 Inicio", Data: "h|" + state.ID}})
	}
	return response
}

func (h *Handler) init(ctx context.Context, args []string) (domain.BotResponse, error) {
	target := ""
	if len(args) > 0 {
		target = args[0]
	}
	workingDir, err := h.resolveInitTarget(ctx, target)
	if err != nil {
		return domain.BotResponse{Text: err.Error()}, nil
	}
	if err := h.server.Start(ctx, workingDir); err != nil {
		return domain.BotResponse{Text: "No se pudo arrancar el servidor: " + err.Error()}, nil
	}
	state, _ := h.state.LoadRuntimeState(ctx)
	state.SessionID = ""
	state.RelativePath = relativeUnderWorkspace(h.workspaceRoot, workingDir)
	if state.WorkspaceRoot == "" {
		state.WorkspaceRoot = h.workspaceRoot
	}
	if err := h.state.SaveRuntimeState(ctx, state); err != nil {
		return domain.BotResponse{}, err
	}
	return domain.BotResponse{Text: fmt.Sprintf("Servidor arrancado en `%s`. Usa /sessions para abrir o crear una sesión.", workingDir)}, nil
}

func (h *Handler) resolveInitTarget(ctx context.Context, override string) (string, error) {
	if override != "" {
		if filepath.IsAbs(override) {
			return "", domain.ErrOutsideWorkspace
		}
		absolute, _, err := h.browser.Resolve(override)
		if err != nil {
			return "", err
		}
		return absolute, nil
	}
	state, err := h.state.LoadRuntimeState(ctx)
	if err != nil {
		return "", err
	}
	if state.WorkspaceRoot != "" && state.RelativePath != "" {
		abs := filepath.Join(state.WorkspaceRoot, state.RelativePath)
		if rel, err := filepath.Rel(state.WorkspaceRoot, abs); err == nil && !strings.HasPrefix(rel, "..") {
			return abs, nil
		}
	}
	if state.ProjectID != "" && state.WorkspaceRoot != "" {
		return state.WorkspaceRoot, nil
	}
	if h.workspaceRoot != "" {
		return h.workspaceRoot, nil
	}
	return "", domain.ErrWorkspaceNotConfigured
}

func (h *Handler) bindServer(ctx context.Context, absolutePath string) (domain.BotResponse, error) {
	if err := h.server.Start(ctx, absolutePath); err != nil {
		return domain.BotResponse{Text: "Carpeta guardada, pero no se pudo rearrancar el servidor: " + err.Error(), Edit: true}, nil
	}
	return domain.BotResponse{
		Text: fmt.Sprintf("Proyecto activo:\n`%s`\n\nServidor reiniciado. Usa /sessions para abrir o crear una sesión.",
			filepath.Base(absolutePath)),
		Edit: true,
	}, nil
}

func relativeUnderWorkspace(workspaceRoot, absolutePath string) string {
	rel, err := filepath.Rel(workspaceRoot, filepath.Clean(absolutePath))
	if err != nil || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." {
		return ""
	}
	if rel == "." {
		return ""
	}
	return filepath.ToSlash(rel)
}

func (h *Handler) status(ctx context.Context) (domain.BotResponse, error) {
	state, err := h.state.LoadRuntimeState(ctx)
	if err != nil {
		return domain.BotResponse{}, err
	}
	project := state.RelativePath
	if project == "" {
		project = "ninguno"
	}
	session := state.SessionID
	if session == "" {
		session = "ninguna"
	}
	if !h.server.StartedSubprocess() {
		return domain.BotResponse{Text: fmt.Sprintf("Servidor: apagado\nProyecto: %s\nSesión: %s\n\nArranca con /init.", project, session)}, nil
	}
	health, err := h.opencode.Health(ctx)
	if err != nil {
		return domain.BotResponse{Text: "OpenCode no responde."}, nil
	}
	return domain.BotResponse{Text: fmt.Sprintf("OpenCode: %t (%s)\nServidor cwd: %s\nProyecto: %s\nSesión: %s",
		health.Healthy, health.Version, h.server.WorkingDir(), project, session)}, nil
}

func (h *Handler) sessions(ctx context.Context, chatID int64, args []string) (domain.BotResponse, error) {
	if !h.server.StartedSubprocess() {
		return domain.BotResponse{Text: "El servidor OpenCode está apagado. Ejecuta /init primero."}, nil
	}
	state, err := h.state.LoadRuntimeState(ctx)
	if err != nil {
		return domain.BotResponse{}, err
	}
	if state.ProjectID == "" {
		return domain.BotResponse{Text: "Selecciona primero un proyecto con /projects."}, nil
	}
	for _, arg := range args {
		switch arg {
		case "new":
			session, err := h.opencode.CreateSession(ctx, "")
			if err != nil {
				return domain.BotResponse{}, err
			}
			state.SessionID = session.ID
			if err := h.state.SaveRuntimeState(ctx, state); err != nil {
				return domain.BotResponse{}, err
			}
			return domain.BotResponse{Text: "Nueva sesión activa: " + session.ID}, nil
		case "current":
			return domain.BotResponse{Text: "Sesión activa: " + orDefault(state.SessionID, "ninguna")}, nil
		}
	}
	if len(args) > 0 {
		sessionID := args[0]
		state.SessionID = sessionID
		if err := h.state.SaveRuntimeState(ctx, state); err != nil {
			return domain.BotResponse{}, err
		}
		return domain.BotResponse{Text: "Sesión activa: " + sessionID}, nil
	}
	all, err := h.opencode.ListSessions(ctx)
	if err != nil {
		return domain.BotResponse{}, err
	}
	activeDir := ""
	if state.WorkspaceRoot != "" {
		activeDir, _ = filepath.Abs(filepath.Join(state.WorkspaceRoot, state.RelativePath))
	}
	sessions := filterSessionsByDirectory(all, activeDir)
	if len(sessions) == 0 {
		return domain.BotResponse{Text: "No hay sesiones abiertas todavía. Usa /sessions new para crear una."}, nil
	}
	response := domain.BotResponse{Text: "Sesiones disponibles:"}
	for _, session := range sessions {
		label := session.Title
		if label == "" {
			label = session.ID
		}
		response.Buttons = append(response.Buttons, []domain.BotButton{{Text: label, Data: fmt.Sprintf("sn|%d|%s", chatID, session.ID)}})
	}
	response.Buttons = append(response.Buttons, []domain.BotButton{{Text: "Nueva sesión", Data: fmt.Sprintf("sn|%d|new", chatID)}})
	return response, nil
}

func filterSessionsByDirectory(sessions []domain.Session, dir string) []domain.Session {
	if dir == "" {
		return sessions
	}
	out := make([]domain.Session, 0, len(sessions))
	for _, s := range sessions {
		if s.Directory == dir {
			out = append(out, s)
		}
	}
	return out
}

func orDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func (h *Handler) diff(ctx context.Context) (domain.BotResponse, error) {
	if !h.server.StartedSubprocess() {
		return domain.BotResponse{Text: "El servidor OpenCode está apagado. Ejecuta /init primero."}, nil
	}
	state, err := h.state.LoadRuntimeState(ctx)
	if err != nil {
		return domain.BotResponse{}, err
	}
	if state.SessionID == "" {
		return domain.BotResponse{Text: "No hay sesión activa."}, nil
	}
	changes, err := h.opencode.FileStatus(ctx, state.SessionID)
	if err != nil {
		return domain.BotResponse{}, err
	}
	if len(changes) == 0 {
		return domain.BotResponse{Text: "No hay cambios."}, nil
	}
	var text strings.Builder
	for _, change := range changes {
		text.WriteString(change.Status + " `" + change.Path + "`\n")
	}
	return domain.BotResponse{Text: text.String()}, nil
}

func (h *Handler) undo(ctx context.Context) (domain.BotResponse, error) {
	if !h.server.StartedSubprocess() {
		return domain.BotResponse{Text: "El servidor OpenCode está apagado. Ejecuta /init primero."}, nil
	}
	state, err := h.state.LoadRuntimeState(ctx)
	if err != nil {
		return domain.BotResponse{}, err
	}
	if state.SessionID == "" {
		return domain.BotResponse{Text: "No hay sesión activa."}, nil
	}
	if err := h.opencode.Revert(ctx, state.SessionID); err != nil {
		return domain.BotResponse{}, err
	}
	return domain.BotResponse{Text: "Último cambio revertido."}, nil
}

func expiredNavigation() domain.BotResponse {
	return domain.BotResponse{Text: "El menú expiró. Usa /projects para abrir uno nuevo.", Edit: true}
}

func navigationError(err error) domain.BotResponse {
	if errors.Is(err, domain.ErrNavigationNotFound) || errors.Is(err, domain.ErrUnauthorizedNavigation) {
		return expiredNavigation()
	}
	return domain.BotResponse{Text: "No se pudo completar la navegación.", Edit: true}
}
