# Design

This document explains how OpenCode Remote is built and why. It complements
`README.md` (user-facing) and `PRODUCT.md` (scope) by recording the
architectural decisions and trade-offs that shaped the code.

## Goals and non-goals

### Goals

- Single Go binary. No GUI, no daemon manager, no app bundle.
- Strict layering: pure domain, swappable adapters, easy testing.
- Strict security by default: workspace-bounded, single-user, no public
  ports.
- Small surface area: only Telegram long polling and `opencode serve` on
  localhost.

### Non-goals

- Multi-user access.
- Multi-chat concurrent flows.
- Multimedia inputs (voice, documents, images).
- Any kind of reverse-proxy or proxy support (yet).
- Hosting `opencode serve` from inside the bot.

## Layered architecture

The backend follows hexagonal / clean architecture. Three concentric rings,
dependency arrows pointing inward only.

```text
        ┌───────────────────────────────┐
        │          adapters             │
        │  telegram · opencode · sqlite │
        │  workspace · config (.env)    │
        └───────────────┬───────────────┘
                        │ implements
        ┌───────────────▼───────────────┐
        │           usecase             │
        │  browser · navigation ·       │
        │  bot handler                  │
        └───────────────┬───────────────┘
                        │ uses
        ┌───────────────▼───────────────┐
        │           domain              │
        │  entities + ports             │
        │  (zero external deps)         │
        └───────────────────────────────┘
```

### `internal/domain`

Pure data and interfaces.

- Entities: `Project`, `Session`, `RuntimeState`, `NavigationState`,
  `DirectoryEntry`, `OpenCodeEvent`, `BotButton`, `BotResponse`.
- Ports: `WorkspaceFS`, `StateRepository`, `NavigationRepository`,
  `OpenCodeClient`, `BotHandler`.
- Errors: `ErrOutsideWorkspace`, `ErrNotDirectory`, `ErrNavigationNotFound`.

The domain package imports nothing from the rest of the project and nothing
external besides the standard library. This is what keeps every test fast
and every adapter swappable.

### `internal/usecase`

Use cases — the rules of the product.

- `WorkspaceBrowser`: walks the workspace, resolves symlinks, enforces the
  "must stay inside the root" invariant.
- `NavigationService`: short-lived per-chat navigation records with
  expiration; only the owning chat can drive the buttons.
- `BotHandler`: command router. Pure dispatch — no knowledge of Telegram
  specifics; only `BotResponse` values come back.

### `internal/adapter`

Adapters that implement the ports and depend on real-world libraries.

- `adapter/opencode`: HTTP client for `http://127.0.0.1:<port>`, with a
  separate SSE client (no timeout) and a parser for `event:` / `data:`
  frames with reconnect-with-backoff.
- `adapter/telegram`: telebot.v3 long polling, whitelist middleware,
  commands, `OnText`, and a single inline-button endpoint that decodes
  `kind|id|payload` callback data.
- `adapter/storage/sqlite`: SQLite repository with WAL mode, single
  connection to avoid locking. Holds `runtime_state` and
  `directory_navigation`.
- `adapter/workspace`: thin wrapper around the standard `os` filesystem
  helpers, exposed as a `WorkspaceFS` port to keep tests hermetic.
- `config`: `.env` loader (godotenv.Overload), parent-directory walk,
  `ENV_FILE` override.

### `cmd/remote-bot/main.go`

The composition root: parse config, initialize adapters, wire up use cases,
start the bot, react to `SIGINT` / `SIGTERM`.

## Trust boundaries

```text
┌─────────────┐  long polling  ┌─────────────┐  REST + SSE  ┌────────────┐
│  Telegram   │ ─────────────► │  remote-bot │ ───────────► │ opencode   │
│   user      │ ◄───────────── │   (Go)      │ ◄─────────── │   serve    │
└─────────────┘   callbacks    └──────┬──────┘              └────────────┘
                                      │
                                      ▼
                                 ┌──────────┐
                                 │  SQLite  │
                                 └──────────┘
```

- **Inbound**: only `api.telegram.org`. No listening sockets.
- **Outbound**: only `127.0.0.1:<OPENCODE_PORT>`.
- **Storage**: a single SQLite file with the runtime state.

## Workspace safety

The whole product hinges on one invariant: every path the bot touches must
resolve to a directory strictly inside `WORKSPACE_ROOT`. This is enforced in
one place — `WorkspaceBrowser.resolve` — and exercised by tests for:

- `..` traversal.
- Absolute paths outside the root.
- Symlinks whose target escapes the root.
- Hidden directories (skipped on listing).
- Symlinks inside the root that themselves point outside (skipped on
  listing).

Navigation is always relative. The handler never receives an absolute path
from Telegram; it only receives a navigation record ID and a relative path
validated against that record.

## Concurrency model

- The bot is single-threaded for Telegram handlers: telebot.v3 runs them in
  goroutines, but every operation that touches shared state goes through
  SQLite, which we serialize with a single connection.
- SSE is consumed in a separate goroutine when wired in; today the project
  ships the SSE adapter and the bridge plumbing but the bot does not yet
  forward events to chat. The adapter is tested in isolation.
- The CLI process shuts down on `SIGINT` / `SIGTERM` via
  `signal.NotifyContext`; the bot stops its long poller and the SQLite
  connection is closed.

## Why Go

- Single static binary, trivial to distribute.
- Strong standard library for HTTP, SSE parsing and context cancellation.
- `modernc.org/sqlite` removes the CGO toolchain dependency so the binary
  remains a single artifact with no system SQLite requirement.
- `gopkg.in/telebot.v3` is a small, focused Telegram library with long
  polling and inline keyboards.

## Why not a SwiftUI menu bar app anymore

The project originally shipped with a SwiftUI `MenuBarExtra` wrapper. That
was removed in favor of a single Go binary for two reasons:

1. Gatekeeper on modern macOS rejects ad-hoc-signed bundles, which makes
   the macOS wrapper impractical to distribute without a paid Apple
   Developer ID.
2. The product is a single-user personal tool — a CLI binary fits the use
   case better and removes an entire frontend surface area from the
   threat model.

## What we keep in mind when adding code

- Anything that touches the trust model (whitelist, workspace validation,
  secret handling) gets tests first.
- New Telegram commands are added in exactly three places: the
  `commands` slice in the adapter, the dispatcher in the bot handler, and
  a unit test.
- Adapters depend inward only; the domain package must remain
  import-free.
- Test coverage on the workspace browser and navigation service is
  non-negotiable. They are the security perimeter.
