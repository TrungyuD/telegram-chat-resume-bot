package bot

import (
	"fmt"
	"strings"

	"github.com/user/telegram-claude-bot/internal/events"
	"github.com/user/telegram-claude-bot/internal/store"
	tele "gopkg.in/telebot.v4"
)

func (b *Bot) handleStart(c tele.Context) error {
	user := b.ensureUser(c)
	if user == nil {
		return c.Send("Failed to create user.")
	}

	if !user.IsWhitelisted {
		events.Bus.Emit(events.EventUserJoined, map[string]any{"telegram_id": user.TelegramID, "username": user.Username})
		return c.Send(fmt.Sprintf("Welcome! Your ID: %s\nYou need to be whitelisted by an admin.", user.TelegramID))
	}

	settings := store.GetEffectiveSettings(user.TelegramID, b.config)
	return c.Send(
		fmt.Sprintf("<b>Welcome to Claude Bot!</b>\n\nModel: %s\nEffort: %s\n\nSend any message to start chatting with Claude.\nUse /help for commands.",
			settings.Model, settings.Effort),
		&tele.SendOptions{ParseMode: tele.ModeHTML},
	)
}

func (b *Bot) handleHelp(c tele.Context) error {
	user := b.ensureUser(c)
	tid := ""
	if user != nil {
		tid = user.TelegramID
	}

	helpText := "<b>Commands</b>\n\n" +
		"<b>Chat</b>\n" +
		"Just send a message to chat with Claude\n" +
		"/ask &lt;question&gt; - Quick Q&amp;A (no tools)\n" +
		"/plan &lt;task&gt; - Plan mode (read-only tools)\n" +
		"/stop - Stop active query\n" +
		"/clear - Clear conversation\n\n" +
		"<b>Settings</b>\n" +
		"/model [name] - View/change AI model\n" +
		"/effort [level] - Set effort (low/medium/high)\n" +
		"/settings - View all settings\n\n" +
		"<b>Data</b>\n" +
		"/project &lt;path&gt; - Set working directory\n" +
		"/rule add|remove|toggle|list - Manage rules\n" +
		"/memory save|get|list|delete|clear - Manage memory\n" +
		"/sessions [switch &lt;id&gt;] - Manage sessions\n" +
		"/file &lt;path&gt; - Send a file\n\n" +
		"<b>Info</b>\n" +
		"/status - Show status\n" +
		"/cost - View usage costs"

	if b.isAdmin(tid) {
		helpText += "\n\n<b>Admin</b>\n" +
			"/admin whitelist|ban|remove|users|stats|rule\n" +
			"/mcp add|remove|toggle|list"
	}

	return c.Send(helpText, &tele.SendOptions{ParseMode: tele.ModeHTML})
}

func (b *Bot) handleClear(c tele.Context) error {
	tid := telegramID(c)
	_ = store.DeactivateSession(tid)
	return c.Send("Conversation cleared. New session will start on next message.")
}

func (b *Bot) handleStop(c tele.Context) error {
	tid := telegramID(c)
	if b.claude.HasActiveQuery(tid) {
		if b.claude.InterruptQuery(tid) {
			return c.Send("Query interrupted.")
		}
		return c.Send("Failed to interrupt query.")
	}
	return c.Send("No active query to stop.")
}

func (b *Bot) handleStatus(c tele.Context) error {
	tid := telegramID(c)
	user, _ := store.GetUser(tid)
	settings := store.GetEffectiveSettings(tid, b.config)
	session, _ := store.GetActiveSession(tid)
	totalCost := store.GetUserTotalCost(tid)
	todayCost := store.GetUserCostToday(tid)

	sessionInfo := "none"
	if session != nil {
		sessionInfo = session.Title
		if sessionInfo == "" {
			sessionInfo = session.SessionID[:min(12, len(session.SessionID))]
		}
	}

	role := "user"
	if user != nil {
		role = user.Role
	}

	return c.Send(
		fmt.Sprintf("<b>Status</b>\nRole: %s\nModel: %s\nEffort: %s\nWorking dir: %s\nSession: %s\nCost today: $%.4f\nCost total: $%.4f\nActive queries: %d",
			role, settings.Model, settings.Effort,
			currentWorkingDir(user, session), sessionInfo, todayCost, totalCost,
			b.claude.GetActiveProcessCount()),
		&tele.SendOptions{ParseMode: tele.ModeHTML},
	)
}

func (b *Bot) handleCost(c tele.Context) error {
	tid := telegramID(c)
	totalCost := store.GetUserTotalCost(tid)
	todayCost := store.GetUserCostToday(tid)

	return c.Send(
		fmt.Sprintf("<b>Usage Costs</b>\nToday: $%.4f\nTotal: $%.4f", todayCost, totalCost),
		&tele.SendOptions{ParseMode: tele.ModeHTML},
	)
}

func normalizeCommandArg(arg string) string {
	return strings.TrimSpace(arg)
}
