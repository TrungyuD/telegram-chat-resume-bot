package bot

import (
	"fmt"
	"strings"

	"github.com/user/telegram-claude-bot/internal/events"
	"github.com/user/telegram-claude-bot/internal/store"
	tele "gopkg.in/telebot.v4"
)

func (b *Bot) handleAdmin(c tele.Context) error {
	args := normalizeCommandArg(c.Message().Payload)
	firstSpace := strings.Index(args, " ")
	sub := args
	rest := ""
	if firstSpace > 0 {
		sub = args[:firstSpace]
		rest = strings.TrimSpace(args[firstSpace+1:])
	}

	switch sub {
	case "whitelist":
		if rest == "" {
			return c.Send("Usage: /admin whitelist <telegram_id>")
		}
		if err := store.SetWhitelist(rest, true); err != nil {
			user := &store.User{
				TelegramID:    rest,
				Role:          "user",
				IsWhitelisted: true,
				CreatedAt:     store.NowUTC(),
			}
			_ = store.CreateUser(user)
		}
		return c.Send(fmt.Sprintf("User %s whitelisted.", rest))

	case "ban":
		if rest == "" {
			return c.Send("Usage: /admin ban <telegram_id>")
		}
		_ = store.SetWhitelist(rest, false)
		return c.Send(fmt.Sprintf("User %s banned.", rest))

	case "remove":
		if rest == "" {
			return c.Send("Usage: /admin remove <telegram_id>")
		}
		_ = store.DeleteUser(rest)
		return c.Send(fmt.Sprintf("User %s removed.", rest))

	case "users":
		users, _ := store.ListAllUsers()
		if len(users) == 0 {
			return c.Send("No users.")
		}
		var sb strings.Builder
		sb.WriteString("<b>Users</b>\n")
		for _, user := range users {
			status := "❌"
			if user.IsWhitelisted {
				status = "✅"
			}
			sb.WriteString(fmt.Sprintf("%s %s (@%s) [%s]\n", status, user.TelegramID, user.Username, user.Role))
		}
		return c.Send(sb.String(), &tele.SendOptions{ParseMode: tele.ModeHTML})

	case "stats":
		stats, _ := store.GetStats()
		costStats := store.GetAllCostStats()
		var sb strings.Builder
		sb.WriteString("<b>Stats</b>\n")
		for key, value := range stats {
			sb.WriteString(fmt.Sprintf("%s: %v\n", key, value))
		}
		sb.WriteString("\n<b>Costs</b>\n")
		for key, value := range costStats {
			sb.WriteString(fmt.Sprintf("%s: $%.4f\n", key, value))
		}
		sb.WriteString(fmt.Sprintf("\nActive queries: %d", b.claude.GetActiveProcessCount()))
		return c.Send(sb.String(), &tele.SendOptions{ParseMode: tele.ModeHTML})

	case "rule":
		return b.handleAdminRule(c, rest)

	case "":
		return c.Send("Usage: /admin whitelist|ban|remove|users|stats|rule")

	default:
		return c.Send("Usage: /admin whitelist|ban|remove|users|stats|rule")
	}
}

func (b *Bot) handleAdminRule(c tele.Context, args string) error {
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
			return c.Send("Usage: /admin rule add <name> | <content>")
		}
		name := strings.TrimSpace(rest[:pipeIdx])
		content := strings.TrimSpace(rest[pipeIdx+1:])
		added, _ := store.AddGlobalRule(name, content)
		if !added {
			return c.Send(fmt.Sprintf("Global rule %q already exists.", name))
		}
		return c.Send(fmt.Sprintf("Global rule %q added.", name))

	case "remove":
		if rest == "" {
			return c.Send("Usage: /admin rule remove <name>")
		}
		removed, _ := store.RemoveGlobalRule(rest)
		if !removed {
			return c.Send(fmt.Sprintf("Global rule %q not found.", rest))
		}
		return c.Send(fmt.Sprintf("Global rule %q removed.", rest))

	case "list":
		rules, _ := store.ListGlobalRules()
		if len(rules) == 0 {
			return c.Send("No global rules.")
		}
		return c.Send("Global rules:\n" + store.FormatRuleList(rules))

	default:
		return c.Send("Usage: /admin rule add|remove|list")
	}
}

func (b *Bot) handleMcp(c tele.Context) error {
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
		parts := strings.SplitN(rest, " ", 3)
		if len(parts) < 3 {
			return c.Send("Usage: /mcp add <name> <type> <config_json>")
		}
		server, err := store.ParseMcpServerConfig(parts[0], parts[1], parts[2])
		if err != nil {
			return c.Send("Invalid MCP config: " + err.Error())
		}
		if err := store.AddMcpServer(server); err != nil {
			return c.Send("Failed to add MCP server.")
		}
		events.Bus.Emit(events.EventMCPChanged, map[string]any{"action": "add", "name": server.Name})
		return c.Send(fmt.Sprintf("MCP server %q added.", server.Name))

	case "remove":
		if rest == "" {
			return c.Send("Usage: /mcp remove <name>")
		}
		removed, _ := store.RemoveMcpServer(rest)
		if !removed {
			return c.Send(fmt.Sprintf("MCP server %q not found.", rest))
		}
		events.Bus.Emit(events.EventMCPChanged, map[string]any{"action": "remove", "name": rest})
		return c.Send(fmt.Sprintf("MCP server %q removed.", rest))

	case "toggle":
		if rest == "" {
			return c.Send("Usage: /mcp toggle <name>")
		}
		found, isActive, err := store.ToggleMcpServer(rest)
		if err != nil {
			return c.Send("Failed to toggle MCP server.")
		}
		if !found {
			return c.Send(fmt.Sprintf("MCP server %q not found.", rest))
		}
		status := "disabled"
		if isActive {
			status = "enabled"
		}
		events.Bus.Emit(events.EventMCPChanged, map[string]any{"action": "toggle", "name": rest})
		return c.Send(fmt.Sprintf("MCP server %q %s.", rest, status))

	case "list", "":
		servers, _ := store.ListMcpServers()
		if len(servers) == 0 {
			return c.Send("No MCP servers configured.")
		}
		return c.Send("MCP Servers:\n" + store.FormatServerList(servers))

	default:
		return c.Send("Usage: /mcp add|remove|toggle|list")
	}
}
