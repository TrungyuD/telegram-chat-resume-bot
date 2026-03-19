# Codebase Summary

## Project Structure

**Total LOC:** ~4,314 lines of Go code

```
telegram-chat-resume-bot/
├── cmd/bot/
│   └── main.go (63 LOC) — Entry point, lifecycle management
├── internal/
│   ├── bot/ — Telegram bot handlers & UI
│   ├── chat/ — Chat orchestration, system prompt, session compaction
│   ├── claude/ — Claude CLI wrapper, rate limiting, types
│   ├── config/ — Global config loading (env + config.json)
│   ├── users/ — User profile CRUD
│   ├── settings/ — Per-user AI settings
│   ├── sessions/ — Session lifecycle management
│   ├── rules/ — Global/user rule management
│   ├── memory/ — Per-user key-value memory
│   ├── mcp/ — MCP server configuration
│   ├── costs/ — Cost tracking
│   ├── logs/ — Activity logging
│   ├── platform/storage/ — File I/O primitives, JSON, Markdown, locking
│   ├── events/ — EventBus pub/sub
│   ├── format/ — Markdown utilities
│   ├── dashboard/ — HTTP REST API + WebSocket
│   └── store/ — Backward-compat shim (delegates to domain packages)
├── scripts/
│   ├── pre-commit — Git pre-commit hook
│   └── docker-entrypoint.sh — Docker entrypoint (validates data dir)
├── Dockerfile — Multi-stage: Go builder + node:22-alpine runtime
├── docker-compose.yml — Service definition with volume, healthcheck
├── .env.docker.example — Docker env var template
└── data/ — Runtime data (gitignored)
```

---

## Package Breakdown

### cmd/bot/ (63 LOC)
**Role:** Application entry point and lifecycle.

**File:** `main.go`

**Responsibilities:**
- Load environment variables from `.env` (optional, env vars take precedence)
- Initialize data directories
- Load global config
- Create and start bot + dashboard server
- Graceful shutdown on SIGINT/SIGTERM

**Key Functions:**
- `main()` — Entry point
- Signal handler cleanup: close bot, save state, exit

---

### internal/bot/ (1,604 LOC)

**Role:** Telegram bot UI layer — commands, inline keyboards, message handlers.

**Files:**
| File | LOC | Purpose |
|------|-----|---------|
| `bot.go` | 134 | Bot struct setup, command registration, middleware |
| `handlers.go` | 962 | 17+ command handlers (/start, /help, /ask, /plan, /settings, /rule, /memory, /sessions, /admin, /mcp, etc.) |
| `callbacks.go` | 160 | Inline keyboard button callbacks (model, effort, thinking toggles) |
| `keyboards.go` | 128 | Inline keyboard builders for settings UI |
| `helpers.go` | 82 | Utilities: format cost, build system prompt, check admin, split messages |
| `middleware.go` | 70 | Auth middleware: whitelist/ban checks |

**Key Types:**
```go
type Bot struct {
    tele *tele.Bot
    claude *claude.Client
    store *store.Store
    config *store.GlobalConfig
    eventBus *events.EventBus
}
```

**Key Handlers:**
| Handler | Trigger | Behavior |
|---------|---------|----------|
| `handleStart` | /start | Welcome msg + current settings |
| `handleHelp` | /help | List all commands |
| `handleMessage` | Any text | Full query (all tools) |
| `handleAsk` | /ask <q> | Q&A without tools |
| `handlePlan` | /plan <task> | Read-only tools (Read, Glob, Grep, Bash) |
| `handleSettings` | /settings | Show inline keyboard for settings |
| `handleModel` | /model [name] | Change AI model |
| `handleEffort` | /effort [level] | Change reasoning effort |
| `handleRule` | /rule add/remove/list/toggle | Manage personal rules |
| `handleMemory` | /memory save/get/list/delete/clear | Manage memories |
| `handleSessions` | /sessions [switch id] | List/switch sessions |
| `handleProject` | /project <path> | Set working directory |
| `handleFile` | /file <path> | Download file to Telegram |
| `handleCost` | /cost | View token costs |
| `handleAdmin` | /admin <cmd> | Admin commands (whitelist, ban, stats, etc.) |
| `handleMcp` | /mcp add/remove/toggle/list | MCP server management |

**Query Modes:**
```go
// Query modes control which tools are available
Full     → all tools enabled
Ask      → no tools (pure Q&A)
Plan     → read-only tools only
```

**Concurrency Patterns:**
- Goroutine per update (telegram updates are sequential per bot)
- Mutex-protected user settings access
- Async event broadcasting via EventBus

---

### internal/claude/ (720 LOC)

**Role:** Claude CLI subprocess wrapper with streaming, rate limiting, session management.

**Files:**
| File | LOC | Purpose |
|------|-----|---------|
| `client.go` | 373 | CLI wrapper, streaming JSON parser, subprocess management |
| `types.go` | 109 | Request/response types (ClaudeOptions, ClaudeResult, ClaudeEvent) |
| `ratelimit.go` | 130 | Sliding window rate limiter with per-user concurrency |
| `session.go` | 108 | Session compaction logic |

**Key Types:**
```go
type Client struct {
    config *store.GlobalConfig
    rateLimiter *RateLimiter
    activeQueries sync.Map // [string]context.CancelFunc
    queryInfo sync.Map     // [string]*QueryInfo
    pendingQuestions sync.Map // [string]interface{}
}

type ClaudeOptions struct {
    Model string
    Effort string
    Thinking string
    SystemPrompt string
    Messages []tele.Message
    AllowedTools []string
    MCP []store.McpServer
    OnPartialResponse func(string) error
    OnToolUse func(string, string) error
    OnThinking func(string) error
    // ... other callbacks
}

type ClaudeResult struct {
    Content string
    SessionID string
    CostUSD float64
    InputTokens int
    OutputTokens int
    Error error
}
```

**Core Functions:**
- `SendToClaude(ctx, options) ClaudeResult` — Main query function
- `parseStreamJSON(reader)` — Stream parser for JSON events
- `compactSession()` — Summarize old messages via Claude Haiku

**Streaming Behavior:**
- CLI called with `--output-format stream-json`
- Events parsed: partial responses, tool use, thinking, questions
- Callbacks invoked for each event (OnPartialResponse, OnToolUse, etc.)
- Partial responses throttled to 3-second edits

**Rate Limiting:**
- `CheckRateLimit(tid)` — Per-user sliding window (requests/minute)
- `CheckConcurrency(tid)` — Max 1 active query per user
- `LimitSystemWide()` — Max 3 concurrent CLI processes

---

### internal/store/ (1,532 LOC)

**Role:** File-based data persistence for all entities.

**Files:**
| File | LOC | Purpose |
|------|-----|---------|
| `store.go` | 308 | Main Store struct, CRUD ops, path management, mutex per-file |
| `config.go` | 216 | Load/save global config, env vars + runtime override |
| `users.go` | 90 | User profiles (ID, name, role, whitelist, banned) |
| `settings.go` | 117 | Per-user settings (model, effort, thinking) |
| `sessions.go` | 157 | Session CRUD, frontmatter parsing, compaction |
| `rules.go` | 213 | Global & personal rules, enable/disable |
| `memory.go` | 127 | Per-user key-value memory |
| `cost.go` | 87 | Cost tracking arrays, calculate totals |
| `mcp.go` | 136 | MCP server configs, enable/disable |
| `logs.go` | 84 | Daily activity logs, append events |

**Key Types:**
```go
type Store struct {
    basePath string
    fileMutexes sync.Map // [string]*sync.Mutex
}

type User struct {
    TelegramID int64
    Name string
    Role string // "user", "admin"
    Whitelisted bool
    Banned bool
    CreatedAt time.Time
}

type UserSettings struct {
    Model string
    Effort string
    Thinking string
}

type SessionMeta struct {
    SessionID string
    TelegramID int64
    Title string
    WorkingDir string
    IsActive bool
    CreatedAt time.Time
    LastUsed time.Time
    Summary string // from compaction
    MessagesCompacted int
}

type SessionMessage struct {
    Role string // "user", "assistant"
    Content string
    Timestamp time.Time
}

type Rule struct {
    Name string
    Content string
    Enabled bool
}

type Memory struct {
    Key string
    Value string
}

type McpServer struct {
    Name string
    Type string // "stdio", "sse", "http"
    Config map[string]interface{}
    Enabled bool
}
```

**Data Files:**
```
data/
├── config.json                      # Global config
├── users/{tid}.json                 # User profile
├── settings/{tid}.json              # Per-user settings
├── rules/global/{name}.json         # Global rules
├── rules/users/{tid}/{name}.json    # Personal rules
├── memory/{tid}/{key}.json          # Memories
├── sessions/{tid}/{sid}.md          # Sessions (Markdown)
├── costs/{tid}.json                 # Cost tracking
├── mcp/{name}.json                  # MCP servers
└── logs/{date}.json                 # Activity logs
```

**Session Markdown Format:**
```markdown
---
session_id: uuid
telegram_id: 123456
title: My Session
working_dir: /home/user/project
is_active: true
created_at: 2025-03-16T10:00:00Z
last_used: 2025-03-16T15:30:00Z
summary: "Summary from compaction..."
messages_compacted: 14
---

## user | 2025-03-16T10:00:00Z
What is Go?

## assistant | 2025-03-16T10:00:05Z
Go is a compiled language...

## user | 2025-03-16T10:01:00Z
Tell me more about goroutines.

## assistant | 2025-03-16T10:01:05Z
Goroutines are lightweight threads...
```

**Concurrency Safety:**
- Per-file `sync.Mutex` via `sync.Map` in Store
- `EnsureMutex(path)` locks before read/write
- Serialized file access prevents data corruption

**Key Functions:**
- `GetUser(tid) User` — Get or create user
- `SaveUser(tid, user)` — Persist user
- `GetActiveSession(tid) SessionMeta` — Get current session
- `SaveSession(tid, meta, messages)` — Persist session (append mode)
- `CompactSession(tid, sid)` — Summarize old messages
- `GetRule(tid, name) Rule` — Get personal or global rule
- `BuildSystemPrompt(tid)` — Combine base + global rules + user rules + memories

---

### internal/dashboard/ (253 LOC)

**Role:** HTTP server for REST API + WebSocket event monitoring.

**File:** `server.go`

**Routes:**
```
GET  /health                 → Health check + uptime
GET  /ws                     → WebSocket upgrade
GET  /api/users              → List all users (auth required)
GET  /api/stats              → System stats (auth required)
GET  /api/config             → Get config (auth required)
POST /api/config             → Set config (auth required)
GET  /api/logs?date=YYYY-MM-DD → Logs by date (auth required)
```

**Authentication:**
- Header: `X-API-Key: key`
- Query: `?api_key=key`
- No auth: /health, /ws (but /ws requires proper message format)

**WebSocket Events (27 types):**
- Message lifecycle: `message_received`, `message_sent`
- SDK lifecycle: `sdk_start`, `sdk_complete`, `sdk_error`
- Tool execution: `sdk_tool_use`, `sdk_tool_result`
- Thinking: `sdk_thinking`
- User actions: `user_joined`, `user_banned`, `user_removed`
- Settings: `setting_changed`
- Sessions: `session_changed`, `session_compacted`
- Data: `rule_changed`, `memory_updated`
- Admin: `mcp_changed`, `config_changed`

**WebSocket Hub:**
- Broadcast to all connected clients via buffered channel (256)
- Goroutine per listener to prevent deadlocks
- Automatic cleanup on client disconnect

---

### internal/events/ (93 LOC)

**Role:** Pub/sub event system for real-time monitoring.

**File:** `eventbus.go`

**Key Types:**
```go
type EventBus struct {
    listeners sync.Map // [int64]*EventListener
}

type EventListener struct {
    events chan Event
}

type Event struct {
    Type string      // event type name
    Data interface{} // event payload
    Timestamp time.Time
}
```

**Key Functions:**
- `Subscribe(id int64) EventListener` — Register listener
- `Unsubscribe(id int64)` — Deregister listener
- `Publish(event Event)` — Broadcast event (async)

**Event Types:**
27 total, covering message flow, API calls, settings changes, session management, cost tracking.

---

### internal/format/ (130 LOC)

**Role:** Utilities for message formatting and conversion.

**File:** `format.go`

**Key Functions:**
- `MarkdownToHTML(markdown string) string` — Convert Markdown to HTML for Telegram
- `SplitMessage(text string, maxLen int) []string` — Split long messages (Telegram 4096 char limit)
- `FormatCost(inputTokens, outputTokens, model string) (costUSD float64, display string)`

---

## Dependency Graph

```
cmd/bot/main.go
    ↓
internal/bot/
    ├→ internal/claude/
    │    ├→ internal/store/
    │    ├→ internal/events/
    │    └→ (executes: claude CLI subprocess)
    ├→ internal/store/
    ├→ internal/events/
    ├→ internal/format/
    └→ internal/dashboard/
         └→ internal/events/

internal/dashboard/
    └→ internal/events/

internal/claude/
    ├→ internal/store/
    └→ internal/events/
```

**No circular dependencies.** Internal packages organized in clean hierarchy.

---

## Key Interfaces & Patterns

### Handler Pattern
Bot commands implemented as callback functions:
```go
func (b *Bot) handleMessage(ctx context.Context, m *tele.Message) error {
    // Validate user (middleware)
    // Check rate limit
    // Get settings
    // Build query options
    // Call claude.SendToClaude()
    // Format response
    // Edit Telegram message
}
```

### Streaming Pattern
Claude CLI responses streamed and edited in real-time:
```go
OnPartialResponse: func(text string) error {
    // Called every 3 seconds during response generation
    return bot.tele.Edit(m, text, &tele.SendOptions{ParseMode: "HTML"})
}
```

### File Locking Pattern
Per-file mutexes for concurrent access:
```go
func (s *Store) EnsureMutex(path string) *sync.Mutex {
    actual, _ := s.fileMutexes.LoadOrStore(path, &sync.Mutex{})
    return actual.(*sync.Mutex)
}

mu := s.EnsureMutex(filePath)
mu.Lock()
defer mu.Unlock()
// Read/write file
```

### EventBus Pattern
Decoupled event publishing:
```go
// Publish from bot handler
eventBus.Publish(Event{Type: "message_received", Data: msg})

// Subscribe in dashboard
listener := eventBus.Subscribe(userID)
for event := range listener.events {
    // Broadcast to WebSocket clients
}
```

---

## Configuration System

### Loading Order
1. Load defaults (hardcoded in code)
2. Override with `.env` file (via godotenv)
3. Override with `data/config.json` at runtime

### Environment Variables
| Variable | Default | Type |
|----------|---------|------|
| TELEGRAM_BOT_TOKEN | — | string |
| ADMIN_TELEGRAM_IDS | — | string (comma-separated) |
| CLAUDE_DEFAULT_MODEL | `claude-sonnet-4-6` | string |
| CLAUDE_DEFAULT_EFFORT | `high` | string |
| CLAUDE_DEFAULT_THINKING | `adaptive` | string |
| CLAUDE_CLI_TIMEOUT_MS | `360000` | int |
| CLAUDE_CONTEXT_MESSAGES | `10` | int |
| MAX_CONCURRENT_CLI_PROCESSES | `3` | int |
| RATE_LIMIT_REQUESTS_PER_MINUTE | `10` | int |
| WEB_PORT | `3000` | int |
| ADMIN_API_KEY | empty | string |
| COMPACT_THRESHOLD | `20` | int |
| COMPACT_KEEP_RECENT | `6` | int |
| COMPACT_ENABLED | `true` | bool |
| ALLOWED_WORKING_DIRS | empty | string (comma-separated) |

---

## Error Handling

### Strategy
- Functions return `error` as last return value
- Errors logged and reported to user
- Graceful fallback on non-critical errors
- Critical errors propagate up

### Common Error Types
- User not whitelisted: early return with message
- Rate limit exceeded: return user message with retry time
- Query timeout: `context.DeadlineExceeded`, user notified
- File I/O: wrapped errors with context
- Invalid session: return empty session

### Timeout Strategy
- CLI process timeout: `CLAUDE_CLI_TIMEOUT_MS` (default 6 minutes)
- Context cancellation: `/stop` command, query interruption
- Recovery: user can retry immediately

---

## Data Flow Diagram

```
Telegram User
    ↓
Bot Handler (validates whitelist/admin status)
    ↓
Rate Limiter (check requests/minute, concurrency)
    ↓
Store.GetActiveSession → Load session metadata + recent messages
    ↓
Store.BuildSystemPrompt → Base + global rules + user rules + memories
    ↓
Claude.SendToClaude(ClaudeOptions)
    ↓
    ├→ Subprocess: claude CLI with --output-format stream-json
    │    ├→ Read session context
    │    ├→ Invoke tools (if allowed by query mode)
    │    └→ Stream responses as JSON events
    │
    ├→ Streaming callbacks triggered:
    │    ├→ OnPartialResponse: edit message every 3s
    │    ├→ OnToolUse: show tool notifications
    │    ├→ OnThinking: display thinking output
    │    └→ OnQuestion: handle interactive questions
    │
    └→ Return ClaudeResult: content, cost, session ID, errors
    ↓
Store.SaveSession → Append new messages to session file
    ↓
Store.RecordCost → Update cost tracking
    ↓
EventBus.Publish → Broadcast events (message_sent, session_changed, etc.)
    ↓
Dashboard WebSocket → Real-time event stream to admin clients
    ↓
Response formatted (Markdown→HTML, split messages) → Sent to Telegram
    ↓
User sees streamed response with 3-second update cadence
    ↓
(Background) Session compaction check → If threshold met, summarize old messages
```

---

## Build & Test

### Build Command
```bash
go build -o dist/telegram-claude-bot ./cmd/bot/
```

### Test Command
```bash
go test ./...
```

### Dev Command (Hot Reload)
```bash
air  # Requires air installed: go install github.com/cosmtrek/air@latest
```

### Dependencies Management
```bash
go mod tidy  # Clean up unused deps
go mod download  # Pre-download all deps
```

---

## Performance Characteristics

| Metric | Target | Notes |
|--------|--------|-------|
| Bot startup | <1s | Load config, init Claude CLI |
| Session load | <100ms | Read JSON/Markdown from disk |
| Query streaming edit | 3s | Throttle Telegram API calls |
| Session compaction | <30s | Background goroutine, Haiku model |
| WebSocket publish | <10ms | Buffered channel, async dispatch |
| Rate limit check | <1ms | Lock-free sync.Map lookup |

---

## Testing Strategy

**Current Status:** v1.0 complete, comprehensive manual testing done.

**Recommended Test Coverage:**
- Unit tests for rate limiter logic
- Unit tests for session compaction algorithm
- Integration tests for Store CRUD operations
- Integration tests for Claude streaming responses
- Load tests for concurrent users
- Stress tests for file locking under contention

