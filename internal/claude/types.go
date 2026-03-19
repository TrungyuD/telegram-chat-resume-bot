package claude

import "sync"

// ImageInput for vision.
type ImageInput struct {
	Base64    string `json:"base64"`
	MediaType string `json:"media_type"`
}

// ClaudeOptions for sendToClaude.
type ClaudeOptions struct {
	TelegramID      string
	WorkingDir      string
	ClaudeSessionID string
	SystemPrompt    string
	Mode            string
	Model           string
	Effort          string
	MaxBudgetUSD    *float64
	TimeoutMs       int

	OnPartialResponse func(text string)
	OnToolUse         func(name string, input string)
	OnThinking        func(text string)
	OnSDKEvent        func(eventType string, data map[string]any)
}

// ClaudeResult returned after query completes.
type ClaudeResult struct {
	Content      string
	SessionID    string
	CostUSD      float64
	InputTokens  int
	OutputTokens int
	Error        string
}

// ToolActivity tracks a tool use during query.
type ToolActivity struct {
	Name   string `json:"name"`
	Input  string `json:"input"`
	Status string `json:"status"`
	Time   string `json:"time"`
}

// QueryActivity tracks an active query with thread-safe Tools access.
type QueryActivity struct {
	TelegramID string         `json:"telegram_id"`
	Model      string         `json:"model"`
	StartTime  string         `json:"start_time"`
	mu         sync.Mutex
	Tools      []ToolActivity `json:"tools"`
}

// AddTool appends a tool activity in a thread-safe manner.
func (qa *QueryActivity) AddTool(t ToolActivity) {
	qa.mu.Lock()
	qa.Tools = append(qa.Tools, t)
	qa.mu.Unlock()
}

// MarkLastToolDone marks the last running tool as done.
func (qa *QueryActivity) MarkLastToolDone() {
	qa.mu.Lock()
	for i := len(qa.Tools) - 1; i >= 0; i-- {
		if qa.Tools[i].Status == "running" {
			qa.Tools[i].Status = "done"
			break
		}
	}
	qa.mu.Unlock()
}

// SnapshotTools returns a copy of the tools slice for safe concurrent reading.
func (qa *QueryActivity) SnapshotTools() []ToolActivity {
	qa.mu.Lock()
	cp := make([]ToolActivity, len(qa.Tools))
	copy(cp, qa.Tools)
	qa.mu.Unlock()
	return cp
}

// CLIEvent represents a parsed event from Claude CLI stream-json output.
type CLIEvent struct {
	Type    string         `json:"type"`
	Subtype string         `json:"subtype"`
	Content string         `json:"content,omitempty"`
	Name    string         `json:"name,omitempty"`
	Input   any            `json:"input,omitempty"`
	Data    map[string]any `json:"data,omitempty"`

	SessionID    string      `json:"session_id,omitempty"`
	Result       string      `json:"result,omitempty"`
	TotalCostUSD float64     `json:"total_cost_usd,omitempty"`
	StopReason   string      `json:"stop_reason,omitempty"`
	Usage        *CLIUsage   `json:"usage,omitempty"`
	Message      *CLIMessage `json:"message,omitempty"`
}

// CLIUsage represents token usage from result event.
type CLIUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// CLIMessage represents assistant message in stream event.
type CLIMessage struct {
	Content []CLIContentBlock `json:"content,omitempty"`
}

// CLIContentBlock represents a content block in assistant message.
type CLIContentBlock struct {
	Type  string `json:"type"`
	Text  string `json:"text,omitempty"`
	Name  string `json:"name,omitempty"`
	ID    string `json:"id,omitempty"`
	Input any    `json:"input,omitempty"`
}
