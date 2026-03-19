package claude

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/TrungyuD/telegram-chat-resume-bot/internal/config"
	"github.com/TrungyuD/telegram-chat-resume-bot/internal/events"
	"github.com/TrungyuD/telegram-chat-resume-bot/internal/platform/storage"
)

// Client wraps the Claude CLI subprocess.
type Client struct {
	config        *config.GlobalConfig
	rateLimiter   *RateLimiter
	activeQueries sync.Map // telegramID -> *exec.Cmd
	queryInfo     sync.Map // telegramID -> *QueryActivity
}

// NewClient creates a new Claude CLI client.
func NewClient(cfg *config.GlobalConfig) *Client {
	return &Client{
		config:      cfg,
		rateLimiter: NewRateLimiter(cfg.RateLimitPerMinute, cfg.MaxConcurrent),
	}
}

// SendToClaude sends a message to Claude CLI and streams the response.
func (c *Client) SendToClaude(ctx context.Context, message string, opts ClaudeOptions) (*ClaudeResult, error) {
	timeoutMs := opts.TimeoutMs
	if timeoutMs <= 0 {
		timeoutMs = c.config.TimeoutMs
	}
	if timeoutMs <= 0 {
		timeoutMs = 300000
	}

	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	args := buildArgs(message, opts)
	log.Printf("[claude] Running: claude %v", args)

	cmd := exec.CommandContext(ctx, "claude", args...)
	if opts.WorkingDir != "" {
		cmd.Dir = opts.WorkingDir
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start claude: %w", err)
	}

	c.activeQueries.Store(opts.TelegramID, cmd)
	activity := &QueryActivity{
		TelegramID: opts.TelegramID,
		Model:      opts.Model,
		StartTime:  storage.NowUTC(),
		Tools:      []ToolActivity{},
	}
	c.queryInfo.Store(opts.TelegramID, activity)

	events.Bus.Emit(events.EventSDKStart, map[string]any{
		"telegram_id": opts.TelegramID,
		"model":       opts.Model,
		"mode":        opts.Mode,
	})

	result := &ClaudeResult{}
	var stderrBuf bytes.Buffer
	stderrDone := make(chan struct{})
	go func() {
		defer close(stderrDone)
		stderrScanner := bufio.NewScanner(stderr)
		for stderrScanner.Scan() {
			stderrBuf.WriteString(stderrScanner.Text())
			stderrBuf.WriteString("\n")
		}
	}()

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 10*1024*1024), 10*1024*1024)
	var accumulatedText string

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}

		var ev CLIEvent
		if err := json.Unmarshal(line, &ev); err != nil {
			continue
		}

		switch ev.Type {
		case "assistant":
			if ev.Message != nil {
				for _, block := range ev.Message.Content {
					switch block.Type {
					case "text":
						if block.Text != "" {
							accumulatedText = block.Text
							if opts.OnPartialResponse != nil {
								opts.OnPartialResponse(accumulatedText)
							}
						}
					case "tool_use":
						inputStr := ""
						if block.Input != nil {
							b, _ := json.Marshal(block.Input)
							inputStr = string(b)
						}
						toolAct := ToolActivity{Name: block.Name, Input: inputStr, Status: "running", Time: storage.NowUTC()}
						activity.AddTool(toolAct)
						if opts.OnToolUse != nil {
							opts.OnToolUse(block.Name, inputStr)
						}
						events.Bus.Emit(events.EventSDKToolUse, map[string]any{
							"telegram_id": opts.TelegramID,
							"tool":        block.Name,
							"input":       inputStr,
						})
					case "thinking":
						if block.Text != "" {
							if opts.OnThinking != nil {
								opts.OnThinking(block.Text)
							}
							events.Bus.Emit(events.EventSDKThinking, map[string]any{"telegram_id": opts.TelegramID})
						}
					}
				}
			}
			if ev.Message == nil {
				switch ev.Subtype {
				case "text":
					if ev.Content != "" {
						accumulatedText += ev.Content
						if opts.OnPartialResponse != nil {
							opts.OnPartialResponse(accumulatedText)
						}
					}
				case "tool_use":
					inputStr := ""
					if ev.Input != nil {
						b, _ := json.Marshal(ev.Input)
						inputStr = string(b)
					}
					toolAct := ToolActivity{Name: ev.Name, Input: inputStr, Status: "running", Time: storage.NowUTC()}
					activity.AddTool(toolAct)
					if opts.OnToolUse != nil {
						opts.OnToolUse(ev.Name, inputStr)
					}
					events.Bus.Emit(events.EventSDKToolUse, map[string]any{
						"telegram_id": opts.TelegramID,
						"tool":        ev.Name,
						"input":       inputStr,
					})
				case "thinking":
					if ev.Content != "" {
						if opts.OnThinking != nil {
							opts.OnThinking(ev.Content)
						}
						events.Bus.Emit(events.EventSDKThinking, map[string]any{"telegram_id": opts.TelegramID})
					}
				}
			}

		case "tool_result":
			activity.MarkLastToolDone()
			events.Bus.Emit(events.EventSDKToolResult, map[string]any{"telegram_id": opts.TelegramID})

		case "result":
			result.SessionID = ev.SessionID
			result.CostUSD = ev.TotalCostUSD
			if ev.Usage != nil {
				result.InputTokens = ev.Usage.InputTokens
				result.OutputTokens = ev.Usage.OutputTokens
			}
			if ev.Result != "" {
				result.Content = ev.Result
			} else {
				result.Content = accumulatedText
			}
		}

		if opts.OnSDKEvent != nil {
			opts.OnSDKEvent(ev.Type, ev.Data)
		}
	}

	<-stderrDone

	if err := cmd.Wait(); err != nil {
		c.activeQueries.Delete(opts.TelegramID)
		c.queryInfo.Delete(opts.TelegramID)

		errMsg := err.Error()
		if stderrBuf.Len() > 0 {
			errMsg = stderrBuf.String()
		}
		events.Bus.Emit(events.EventSDKError, map[string]any{
			"telegram_id": opts.TelegramID,
			"error":       errMsg,
		})
		log.Printf("[claude] Error: %s | Stderr: %s", err.Error(), stderrBuf.String())
		return nil, fmt.Errorf("claude exited: %s", errMsg)
	}

	c.activeQueries.Delete(opts.TelegramID)
	c.queryInfo.Delete(opts.TelegramID)
	if result.Content == "" {
		result.Content = accumulatedText
	}

	events.Bus.Emit(events.EventSDKComplete, map[string]any{
		"telegram_id":   opts.TelegramID,
		"cost_usd":      result.CostUSD,
		"input_tokens":  result.InputTokens,
		"output_tokens": result.OutputTokens,
		"session_id":    result.SessionID,
	})
	return result, nil
}

// buildArgs constructs CLI args for the claude command.
func buildArgs(message string, opts ClaudeOptions) []string {
	args := []string{"-p", message, "--output-format", "stream-json", "--verbose"}
	if opts.Model != "" {
		args = append(args, "--model", opts.Model)
	}
	if effort := supportedEffort(opts.Effort); effort != "" {
		args = append(args, "--effort", effort)
	}
	if opts.MaxBudgetUSD != nil && *opts.MaxBudgetUSD > 0 {
		args = append(args, "--max-budget-usd", strconv.FormatFloat(*opts.MaxBudgetUSD, 'f', -1, 64))
	}
	if opts.ClaudeSessionID != "" && strings.Contains(opts.ClaudeSessionID, "-") {
		args = append(args, "--resume", opts.ClaudeSessionID)
	}
	if opts.SystemPrompt != "" {
		args = append(args, "--system-prompt", opts.SystemPrompt)
	}

	switch opts.Mode {
	case "ask":
		args = append(args, "--allowedTools", "")
	case "plan":
		args = append(args, "--allowedTools", "Read,Glob,Grep,Bash(read-only)")
	default:
		// Full mode: allow common dev tools but not unrestricted shell access
		args = append(args, "--allowedTools", "Read,Write,Edit,MultiEdit,Glob,Grep,Bash")
	}
	return args
}

func supportedEffort(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "low", "medium", "high":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return ""
	}
}

// InterruptQuery gracefully stops the active query for the given telegramID.
// Sends SIGTERM first, waits up to 3 seconds, then SIGKILL if still alive.
func (c *Client) InterruptQuery(telegramID string) bool {
	v, ok := c.activeQueries.Load(telegramID)
	if !ok {
		return false
	}
	cmd := v.(*exec.Cmd)
	if cmd.Process != nil {
		// Try graceful termination first
		_ = cmd.Process.Signal(syscall.SIGTERM)

		// Wait up to 3 seconds for process to exit
		done := make(chan struct{})
		go func() {
			_, _ = cmd.Process.Wait()
			close(done)
		}()
		select {
		case <-done:
			// Process exited gracefully
		case <-time.After(3 * time.Second):
			// Force kill after timeout
			_ = cmd.Process.Kill()
		}
	}
	c.activeQueries.Delete(telegramID)
	c.queryInfo.Delete(telegramID)
	c.rateLimiter.MarkInactive(telegramID)
	return true
}

// HasActiveQuery returns true if the telegramID has an active query.
func (c *Client) HasActiveQuery(telegramID string) bool {
	_, ok := c.activeQueries.Load(telegramID)
	return ok
}

// GetActiveProcessCount returns the number of active queries.
func (c *Client) GetActiveProcessCount() int {
	count := 0
	c.activeQueries.Range(func(_, _ any) bool {
		count++
		return true
	})
	return count
}

// GetActiveQueryInfo returns all active query activities with thread-safe tool snapshots.
func (c *Client) GetActiveQueryInfo() []*QueryActivity {
	var result []*QueryActivity
	c.queryInfo.Range(func(_, v any) bool {
		if qa, ok := v.(*QueryActivity); ok {
			snapshot := &QueryActivity{
				TelegramID: qa.TelegramID,
				Model:      qa.Model,
				StartTime:  qa.StartTime,
				Tools:      qa.SnapshotTools(),
			}
			result = append(result, snapshot)
		}
		return true
	})
	return result
}

// GetRateLimiter returns the rate limiter.
func (c *Client) GetRateLimiter() *RateLimiter {
	return c.rateLimiter
}
