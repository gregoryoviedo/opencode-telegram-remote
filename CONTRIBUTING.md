# Contributing

Thanks for your interest in OpenCode Remote.

## Workflow

1. Fork and create a feature branch.
2. Run `go test ./...` and `go vet ./...` before pushing.
3. Open a pull request against `main`.

## Code style

- Go: `gofmt`, `go vet`, `go test`.
- No secrets in commits. The bot token must come from `.env` at runtime.

## Adding a new Telegram command

1. Add the command string to `commands` in `internal/adapter/telegram/bot.go`.
2. Add the handling case to `BotHandler.HandleCommand` in
   `internal/usecase/bot_handler.go`.
3. Cover the new flow with a unit or integration test in
   `internal/usecase`.

## Releasing

Just tag and commit. Users build the binary themselves with `go build`.
