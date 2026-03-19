package bot

import (
	"fmt"
	"strings"

	"github.com/TrungyuD/telegram-chat-resume-bot/internal/events"
	"github.com/TrungyuD/telegram-chat-resume-bot/internal/format"
	"github.com/TrungyuD/telegram-chat-resume-bot/internal/store"
	tele "gopkg.in/telebot.v4"
)

func (b *Bot) handleRule(c tele.Context) error {
	tid := telegramID(c)
	args := normalizeCommandArg(c.Message().Payload)

	firstSpace := strings.Index(args, " ")
	sub := args
	rest := ""
	if firstSpace > 0 {
		sub = args[:firstSpace]
		rest = strings.TrimSpace(args[firstSpace+1:])
	}

	switch sub {
	case "add":
		pipeIdx := strings.Index(rest, "|")
		if pipeIdx == -1 {
			return c.Send("Usage: /rule add <name> | <content>")
		}
		name := strings.TrimSpace(rest[:pipeIdx])
		content := strings.TrimSpace(rest[pipeIdx+1:])
		if name == "" || content == "" {
			return c.Send("Usage: /rule add <name> | <content>")
		}
		added, err := store.AddUserRule(tid, name, content)
		if err != nil {
			return c.Send("Failed to add rule.")
		}
		if !added {
			return c.Send(fmt.Sprintf("Rule %q already exists.", name))
		}
		events.Bus.Emit(events.EventRuleChanged, map[string]any{"telegram_id": tid, "action": "add", "name": name})
		return c.Send(fmt.Sprintf("Rule %q added.", name))

	case "remove":
		if rest == "" {
			return c.Send("Usage: /rule remove <name>")
		}
		removed, err := store.RemoveUserRule(tid, rest)
		if err != nil {
			return c.Send("Failed to remove rule.")
		}
		if !removed {
			return c.Send(fmt.Sprintf("Rule %q not found.", rest))
		}
		events.Bus.Emit(events.EventRuleChanged, map[string]any{"telegram_id": tid, "action": "remove", "name": rest})
		return c.Send(fmt.Sprintf("Rule %q removed.", rest))

	case "toggle":
		if rest == "" {
			return c.Send("Usage: /rule toggle <name>")
		}
		found, isActive, err := store.ToggleUserRule(tid, rest)
		if err != nil {
			return c.Send("Failed to toggle rule.")
		}
		if !found {
			return c.Send(fmt.Sprintf("Rule %q not found.", rest))
		}
		status := "disabled"
		if isActive {
			status = "enabled"
		}
		events.Bus.Emit(events.EventRuleChanged, map[string]any{"telegram_id": tid, "action": "toggle", "name": rest})
		return c.Send(fmt.Sprintf("Rule %q %s.", rest, status))

	case "list", "":
		rules, _ := store.ListUserRules(tid)
		if len(rules) == 0 {
			return c.Send("No personal rules. Use /rule add <name> | <content>")
		}
		return c.Send("Your rules:\n" + store.FormatRuleList(rules))

	default:
		return c.Send("Usage: /rule add|remove|toggle|list")
	}
}

func (b *Bot) handleMemory(c tele.Context) error {
	tid := telegramID(c)
	args := normalizeCommandArg(c.Message().Payload)

	firstSpace := strings.Index(args, " ")
	sub := args
	rest := ""
	if firstSpace > 0 {
		sub = args[:firstSpace]
		rest = strings.TrimSpace(args[firstSpace+1:])
	}

	switch sub {
	case "save":
		parts := strings.SplitN(rest, " ", 2)
		if len(parts) < 2 {
			return c.Send("Usage: /memory save <key> <value>")
		}
		if err := store.SetMemory(tid, parts[0], parts[1], "general"); err != nil {
			return c.Send("Failed to save memory.")
		}
		events.Bus.Emit(events.EventMemoryUpdated, map[string]any{"telegram_id": tid, "key": parts[0]})
		return c.Send(fmt.Sprintf("Memory %q saved.", parts[0]))

	case "get":
		if rest == "" {
			return c.Send("Usage: /memory get <key>")
		}
		mem, err := store.GetMemory(tid, rest)
		if err != nil || mem == nil {
			return c.Send(fmt.Sprintf("Memory %q not found.", rest))
		}
		return c.Send(fmt.Sprintf("<b>%s</b>\n%s", format.EscapeHTML(mem.Key), format.EscapeHTML(mem.Value)),
			&tele.SendOptions{ParseMode: tele.ModeHTML})

	case "list":
		memories, _ := store.ListMemory(tid)
		if len(memories) == 0 {
			return c.Send("No memories. Use /memory save <key> <value>")
		}
		return c.Send("Your memories:\n" + store.FormatMemoryList(memories))

	case "delete":
		if rest == "" {
			return c.Send("Usage: /memory delete <key>")
		}
		deleted, err := store.DeleteMemory(tid, rest)
		if err != nil {
			return c.Send("Failed to delete memory.")
		}
		if !deleted {
			return c.Send(fmt.Sprintf("Memory %q not found.", rest))
		}
		events.Bus.Emit(events.EventMemoryUpdated, map[string]any{"telegram_id": tid, "key": rest, "action": "delete"})
		return c.Send(fmt.Sprintf("Memory %q deleted.", rest))

	case "clear":
		count, _ := store.ClearMemory(tid)
		return c.Send(fmt.Sprintf("Cleared %d memories.", count))

	case "":
		return c.Send("Usage: /memory save|get|list|delete|clear")

	default:
		return c.Send("Usage: /memory save|get|list|delete|clear")
	}
}
