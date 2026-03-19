# CLAUDE.md

This file provides guidance to Claude Code when working with code in this repository.

## Project Overview

**telegram-chat-resume-bot** — Telegram bot providing AI coding assistant capabilities to whitelisted users. Built in **Go**, supports multiple AI CLI backends (Claude, CCS, Codex, and others) via subprocess integration. Features per-user model/effort/thinking settings, rule system, memory persistence, session management (Markdown-based), image vision, MCP server config.

## Development Commands

```bash
# Run in development
go run ./cmd/bot/

# Build binary
go build -o dist/telegram-chat-resume-bot ./cmd/bot/

# Run built binary
./dist/telegram-chat-resume-bot

# Run tests
go test ./...

# Tidy dependencies
go mod tidy
```

## Architecture

**Single-process, multi-goroutine design.** One Go binary runs:

1. **Telegram Bot** (`internal/bot/`) — telebot v4 in long-polling mode. 17+ commands, inline keyboard callbacks, photo/document uploads.
2. **AI CLI Integration** (`internal/claude/`) — Wraps AI CLI subprocesses (Claude, CCS, Codex, etc.) with streaming JSON output. Per-user settings, rate limiting, session compaction.
3. **Chat Orchestration** (`internal/chat/`) — System prompt assembly, session compaction workflow.
4. **Dashboard** (`internal/dashboard/`) — Admin HTTP API + WebSocket for real-time monitoring.

**Data Storage:** JSON files + Markdown (no database).

- `data/users/{telegram_id}.json` — user profiles
- `data/settings/{telegram_id}.json` — per-user settings
- `data/rules/global/{name}.json`, `data/rules/users/{tid}/{name}.json` — rules
- `data/memory/{tid}/{key}.json` — user memories
- `data/sessions/{tid}/{session_id}.md` — sessions as Markdown with YAML frontmatter
- `data/costs/{tid}.json` — cost tracking arrays
- `data/mcp/{name}.json` — MCP server configs
- `data/logs/{date}.json` — daily activity logs
- `data/config.json` — global config overrides

**Data flow:** Bot handler → AI CLI subprocess with streaming → callbacks update Telegram message → events broadcast via EventBus

## Key Technical Details

- **Runtime is Go.** Uses standard library + telebot v4 + chi + coder/websocket.
- **AI CLI backends** called via `os/exec` with streaming output. Supports session resume, tool allowlists, timeout. Currently wraps Claude CLI; designed to support CCS, Codex, and other CLI tools.
- **No database.** All data stored as JSON files with per-path mutex locking for concurrent safety.
- **Sessions** stored as Markdown files with YAML frontmatter. Messages appended via `os.O_APPEND`.
- **Session compaction** summarizes old messages, writes summary to frontmatter.
- **Config** loaded from env vars (`.env`), overridable via `data/config.json`.

## Bot Commands

| Command                              | Description                                |
| ------------------------------------ | ------------------------------------------ |
| `/model [name]`                      | View/change AI model                       |
| `/effort [level]`                    | Set reasoning effort (low/medium/high/max) |
| `/thinking [mode]`                   | Set thinking mode (on/off/adaptive)        |
| `/settings`                          | View all settings with inline keyboard     |
| `/rule add/list/remove/toggle`       | Manage personal rules                      |
| `/memory save/get/list/delete/clear` | Manage per-user memory                     |
| `/sessions [switch <id>]`            | List/switch sessions                       |
| `/ask <question>`                    | Quick Q&A without tools                    |
| `/plan <task>`                       | Plan mode (read-only tools)                |
| `/stop`                              | Interrupt active query                     |
| `/cost`                              | View usage costs                           |
| `/mcp add/remove/toggle/list`        | MCP server management (admin)              |
| `/admin rule add/remove/list`        | Global rule management (admin)             |

## File Map

| Package                      | Role                                                            |
| ---------------------------- | --------------------------------------------------------------- |
| `cmd/bot/main.go`            | Entry point, lifecycle                                          |
| `internal/bot/`              | Telegram transport — handlers, callbacks, keyboards, middleware |
| `internal/chat/`             | Chat orchestration — system prompt, session compaction          |
| `internal/claude/`           | AI CLI wrapper, types, rate limiting                            |
| `internal/config/`           | Global config loading (env + config.json)                       |
| `internal/users/`            | User profile CRUD                                               |
| `internal/settings/`         | Per-user AI settings                                            |
| `internal/sessions/`         | Session lifecycle management                                    |
| `internal/rules/`            | Global/user rule management                                     |
| `internal/memory/`           | Per-user key-value memory                                       |
| `internal/mcp/`              | MCP server configuration                                        |
| `internal/costs/`            | Cost tracking                                                   |
| `internal/logs/`             | Activity logging                                                |
| `internal/platform/storage/` | File I/O primitives — JSON, Markdown, locking                   |
| `internal/events/`           | EventBus pub/sub                                                |
| `internal/format/`           | Markdown→HTML converter, message splitting                      |
| `internal/dashboard/`        | Admin HTTP API + WebSocket hub                                  |
| `internal/store/`            | Backward-compat shim (delegates to domain packages)             |
| `data/`                      | Runtime data files (gitignored)                                 |
