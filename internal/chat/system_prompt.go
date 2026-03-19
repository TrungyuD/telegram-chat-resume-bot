package chat

import (
	"fmt"
	"strings"

	"github.com/TrungyuD/telegram-chat-resume-bot/internal/memory"
	"github.com/TrungyuD/telegram-chat-resume-bot/internal/rules"
)

// BuildSystemPrompt assembles the system prompt from rules and memories.
func BuildSystemPrompt(telegramID string) (string, error) {
	var sb strings.Builder

	sb.WriteString("You are a Telegram bot assistant powered by Claude AI.\n")
	sb.WriteString("Keep responses concise and well-formatted. Use markdown where appropriate.\n")
	sb.WriteString("You are running inside Telegram, so avoid overly long responses.\n\n")

	sb.WriteString("## File & Image Delivery\n")
	sb.WriteString("You can send files and images directly in the chat. ")
	sb.WriteString("If you generate or have access to a file or image, you may deliver it directly to the user.\n\n")

	globalRules, userRules, err := rules.GetActiveRules(telegramID)
	if err != nil {
		return "", fmt.Errorf("failed to get active rules: %w", err)
	}

	if len(globalRules) > 0 {
		sb.WriteString("## Global Rules\n")
		for _, r := range globalRules {
			fmt.Fprintf(&sb, "- **%s**: %s\n", r.Name, r.Content)
		}
		sb.WriteString("\n")
	}

	if len(userRules) > 0 {
		sb.WriteString("## Personal Rules\n")
		for _, r := range userRules {
			fmt.Fprintf(&sb, "- **%s**: %s\n", r.Name, r.Content)
		}
		sb.WriteString("\n")
	}

	memories, err := memory.ListMemory(telegramID)
	if err == nil && len(memories) > 0 {
		sb.WriteString("## User Memories\n")
		for _, m := range memories {
			fmt.Fprintf(&sb, "- **%s**: %s\n", m.Key, m.Value)
		}
		sb.WriteString("\n")
	}

	return sb.String(), nil
}
