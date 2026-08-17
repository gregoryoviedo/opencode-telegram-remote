# Product

This document defines what OpenCode Remote is, what it does today, what it
deliberately does not do, and where it is going next.

## Vision

OpenCode Remote turns Telegram into a thin remote control for a local
`opencode serve` instance. The bot is the only surface you interact with on
your phone; OpenCode keeps doing the work locally on your Mac.

## Target scenario

1. You work on a project locally with OpenCode (TUI or desktop).
2. You leave the computer.
3. Later, from your phone, you open the bot in Telegram.
4. You pick a project inside your workspace, switch to or create a session,
   and send prompts.
5. You check progress, run `/diff` and `/undo`, or jump back into `/sessions`.
6. When you return home, the local machine has already done the work.

## Functional scope

### Implemented

- Long polling against `api.telegram.org` with a strict `ALLOWED_CHAT_ID`
  whitelist.
- Recursive workspace navigation limited to `WORKSPACE_ROOT`; traversal and
  escaping symlinks are rejected.
- Project selection by recursive folder picker.
- Session list, switching and inline creation.
- `/status`, `/projects`, `/sessions`, `/diff`, `/changes`, `/undo`.
- Free-form text prompts forwarded to the active session.
- SQLite-backed runtime state (workspace, project, session, navigation).
- Configuration loaded from `.env` (with parent directory walk and
  `ENV_FILE` override).
- Local `opencode serve` lifecycle: the bot can launch, restart, and shut
  down the subprocess for you via `/init`, `/projects → Usar esta carpeta`
  and `OPENCODE_AUTOSTART`.
- `TELEGRAM_PROXY_URL` and `TELEGRAM_API_ROOT` for restricted networks.
- Clean shutdown on `SIGINT` / `SIGTERM`.

### Out of scope (today)

- Any GUI / menu bar app — the project is intentionally CLI-only.
- Multi-user access. The whitelist is a single ID.
- Concurrent interactive flows. One Telegram chat at a time.
- Multimedia (voice, image, document attachments).
- Scheduled tasks.
- i18n.

## Trust boundaries

- The Telegram Bot API is the only network surface the bot depends on.
- `opencode serve` is assumed to be reachable on `127.0.0.1:<OPENCODE_PORT>`.
- The bot token and chat ID live in `.env` (or shell env), never in SQLite.
- SQLite stores the active project, active session and short-lived
  navigation state only.

## Open task list

Items in priority order, intentionally small and incremental:

1. `/new`, `/abort`, `/detach` for explicit session lifecycle.
2. `/messages` listing prior prompts with **Revert** and **Fork** actions.
3. Persistent reply keyboard with the most common actions.
4. Auto-restart of `opencode serve` when health checks fail (localhost only).
5. Pinned live status message (project, session, model, changed files).
6. Scheduled tasks (`/task`, `/tasklist`).
7. Voice transcription via a Whisper-compatible API (opt-in).
8. `TELEGRAM_FORCE_IPV4` and richer proxy options for restricted networks
   (today only `TELEGRAM_PROXY_URL` is supported).

## Change policy

Each item on the task list is small enough to land in one focused change
with tests. Anything that touches the trust model (whitelist, workspace
validation, persistence of secrets) requires a focused review and a
documented test.

Items that change the interaction model (e.g. multi-user, multi-chat
parallel flows) are explicitly out of scope for the moment. See
`docs/DESIGN.md` for the rationale.
