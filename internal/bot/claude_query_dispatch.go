package bot

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/user/telegram-claude-bot/internal/chat"
	"github.com/user/telegram-claude-bot/internal/claude"
	"github.com/user/telegram-claude-bot/internal/events"
	"github.com/user/telegram-claude-bot/internal/format"
	"github.com/user/telegram-claude-bot/internal/store"
	tele "gopkg.in/telebot.v4"
)

func (b *Bot) sendToClaude(c tele.Context, message, mode string) error {
	return b.sendToClaudeWithImages(c, message, nil, mode)
}

func (b *Bot) sendToClaudeWithImages(c tele.Context, message string, images []claude.ImageInput, mode string) error {
	tid := telegramID(c)
	user, _ := store.GetUser(tid)

	rlResult := b.rateLimiter.CheckRateLimit(tid)
	if !rlResult.Allowed {
		return c.Send(fmt.Sprintf("Rate limited: %s\nRetry in %d seconds.", rlResult.Reason, rlResult.RetryAfterSec))
	}

	b.rateLimiter.MarkActive(tid)
	defer b.rateLimiter.MarkInactive(tid)

	settings := store.GetEffectiveSettings(tid, b.config)
	session, _ := store.GetActiveSession(tid)
	workingDir := currentWorkingDir(user, session)
	if session == nil {
		sessionID := fmt.Sprintf("%d", time.Now().UnixNano())
		meta := &store.SessionMeta{
			SessionID:  sessionID,
			TelegramID: tid,
			Title:      truncate(message, 50),
			WorkingDir: workingDir,
			IsActive:   true,
			CreatedAt:  store.NowUTC(),
			LastUsed:   store.NowUTC(),
		}
		_ = store.SaveSession(meta)
		session = meta
		workingDir = meta.WorkingDir
	}

	systemPrompt, _ := chat.BuildSystemPrompt(tid)
	events.Bus.Emit(events.EventMessageReceived, map[string]any{
		"telegram_id": tid,
		"message":     truncate(message, 100),
	})

	sessionPath := store.GetSessionPath(tid, session.SessionID)
	_ = store.AppendSessionMessage(sessionPath, "user", message)

	placeholder, err := b.tele.Send(c.Chat(), "Processing...")
	if err != nil {
		return c.Send("Failed to send message.")
	}

	if len(images) > 0 {
		log.Printf("[bot] image inputs received for %s but current CLI flow does not attach binary image payloads", tid)
	}

	opts := claude.ClaudeOptions{
		TelegramID:      tid,
		WorkingDir:      workingDir,
		ClaudeSessionID: resumeClaudeSessionID(session),
		Mode:            mode,
		Model:           settings.Model,
		Effort:          settings.Effort,
		MaxBudgetUSD:    settings.MaxBudgetUSD,
		TimeoutMs:       b.config.TimeoutMs,
		SystemPrompt:    systemPrompt,
	}

	var editMu sync.Mutex
	lastEdit := time.Now()
	var lastText string

	opts.OnPartialResponse = func(text string) {
		editMu.Lock()
		defer editMu.Unlock()
		lastText = text
		if time.Since(lastEdit) <= 3*time.Second {
			return
		}
		displayText := text
		if len(displayText) > 4000 {
			displayText = displayText[len(displayText)-4000:]
		}
		_, _ = b.tele.Edit(placeholder, displayText)
		lastEdit = time.Now()
	}

	opts.OnToolUse = func(name string, input string) {
		editMu.Lock()
		defer editMu.Unlock()
		toolMsg := fmt.Sprintf("🔧 Using: %s", name)
		if lastText != "" {
			toolMsg = lastText + "\n\n" + toolMsg
		}
		if len(toolMsg) > 4000 {
			toolMsg = toolMsg[len(toolMsg)-4000:]
		}
		_, _ = b.tele.Edit(placeholder, toolMsg)
	}

	ctx := context.Background()
	result, err := b.claude.SendToClaude(ctx, message, opts)
	if err != nil {
		_, _ = b.tele.Edit(placeholder, "Error: "+err.Error())
		return nil
	}

	if result.SessionID != "" && result.SessionID != session.ClaudeSessionID {
		session.ClaudeSessionID = result.SessionID
		_ = store.SaveSession(session)
	}
	if result.Content != "" {
		_ = store.AppendSessionMessage(sessionPath, "assistant", result.Content)
	}
	if result.CostUSD > 0 {
		_ = store.AddCostRecord(tid, &store.CostRecord{
			SessionID:    session.SessionID,
			CostUSD:      result.CostUSD,
			InputTokens:  result.InputTokens,
			OutputTokens: result.OutputTokens,
			Model:        settings.Model,
			CreatedAt:    store.NowUTC(),
		})
	}

	_ = store.UpdateSessionLastUsed(tid, session.SessionID)
	go func() {
		_ = chat.CompactSessionIfNeeded(b.config, tid, session.SessionID)
	}()

	finalText := result.Content
	if finalText == "" {
		finalText = "(No response)"
	}
	htmlText := format.MarkdownToHTML(finalText)
	events.Bus.Emit(events.EventMessageSent, map[string]any{
		"telegram_id": tid,
		"length":      len(finalText),
		"cost":        result.CostUSD,
	})

	_, editErr := b.tele.Edit(placeholder, htmlText, &tele.SendOptions{ParseMode: tele.ModeHTML})
	if editErr != nil {
		_, editErr = b.tele.Edit(placeholder, finalText)
		if editErr != nil {
			_ = b.tele.Delete(placeholder)
			return b.sendLong(c, htmlText, tele.ModeHTML)
		}
	}
	return nil
}

func userWorkingDir(user *store.User) string {
	if user != nil && user.WorkingDirectory != "" {
		return user.WorkingDirectory
	}
	wd, _ := os.Getwd()
	return wd
}

func currentWorkingDir(user *store.User, session *store.SessionMeta) string {
	if session != nil && session.WorkingDir != "" {
		return session.WorkingDir
	}
	return userWorkingDir(user)
}

func resumeClaudeSessionID(session *store.SessionMeta) string {
	if session == nil {
		return ""
	}
	if session.ClaudeSessionID != "" {
		return session.ClaudeSessionID
	}
	if session.SessionID != "" && containsDash(session.SessionID) {
		return session.SessionID
	}
	return ""
}

func containsDash(value string) bool {
	for _, ch := range value {
		if ch == '-' {
			return true
		}
	}
	return false
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
