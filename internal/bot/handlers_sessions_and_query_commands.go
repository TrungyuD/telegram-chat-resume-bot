package bot

import (
	"errors"
	"fmt"
	"strings"

	"github.com/user/telegram-claude-bot/internal/events"
	"github.com/user/telegram-claude-bot/internal/store"
	tele "gopkg.in/telebot.v4"
)

func (b *Bot) handleSessions(c tele.Context) error {
	tid := telegramID(c)
	args := normalizeCommandArg(c.Message().Payload)

	if strings.HasPrefix(args, "switch ") {
		sessionID := strings.TrimSpace(strings.TrimPrefix(args, "switch "))
		if sessionID == "" {
			return c.Send("Usage: /sessions switch <id>")
		}
		if err := store.SwitchSession(tid, sessionID); err != nil {
			if errors.Is(err, store.ErrSessionNotFound) {
				return c.Send(fmt.Sprintf("Session %q not found.", sessionID))
			}
			return c.Send("Failed to switch session.")
		}
		session, err := store.GetSession(tid, sessionID)
		if err != nil {
			return c.Send("Failed to load session metadata.")
		}
		if session.WorkingDir != "" {
			if err := store.SetWorkingDir(tid, session.WorkingDir); err != nil {
				return c.Send("Switched session, but failed to sync working directory.")
			}
		}
		events.Bus.Emit(events.EventSessionChanged, map[string]any{"telegram_id": tid, "session_id": sessionID})
		return c.Send(fmt.Sprintf("Switched to session: %s", sessionID))
	}

	sessions, _ := store.ListSessions(tid)
	if len(sessions) == 0 {
		return c.Send("No sessions. Start chatting to create one.")
	}

	return c.Send("Your sessions:", SessionList(sessions))
}

func (b *Bot) handleAsk(c tele.Context) error {
	question := normalizeCommandArg(c.Message().Payload)
	if question == "" {
		return c.Send("Usage: /ask <question>")
	}
	return b.sendToClaude(c, question, "ask")
}

func (b *Bot) handlePlan(c tele.Context) error {
	task := normalizeCommandArg(c.Message().Payload)
	if task == "" {
		return c.Send("Usage: /plan <task>")
	}
	return b.sendToClaude(c, task, "plan")
}
