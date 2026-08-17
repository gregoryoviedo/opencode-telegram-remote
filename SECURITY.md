# Security

## Trust model

OpenCode Remote is designed to be used by a single person. The Telegram
bot accepts commands only from the configured `ALLOWED_CHAT_ID` and rejects
any callback that does not match that chat ID.

## What the bot can do

- Navigate directories inside the configured workspace.
- Create OpenCode sessions, send prompts, request `/diff`, `/changes`,
  `/undo`.
- It cannot navigate outside the workspace.
- It cannot write to arbitrary files.
- It cannot execute arbitrary shell commands received over Telegram.

## Workspace safety

- All paths are stored as relative paths inside the workspace.
- Symlinks are resolved and verified to stay inside the workspace.
- Absolute paths and traversal attempts are rejected.

## Storage

- The bot token and chat ID are stored in `.env` (or in shell env vars).
- SQLite stores only the active workspace, project, session and ephemeral
  navigation state.
- No prompt contents are stored.

## Distribution

This project is distributed as a single Go binary that the user compiles
themselves from source. There is no signed binary, no Apple Developer ID
required, and no Gatekeeper involvement.
