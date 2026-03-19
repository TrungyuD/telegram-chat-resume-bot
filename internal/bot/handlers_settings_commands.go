package bot

import (
	"fmt"
	"strings"

	"github.com/user/telegram-claude-bot/internal/events"
	"github.com/user/telegram-claude-bot/internal/store"
	tele "gopkg.in/telebot.v4"
)

func (b *Bot) handleModel(c tele.Context) error {
	tid := telegramID(c)
	arg := normalizeCommandArg(c.Message().Payload)

	if arg != "" {
		resolved := store.ResolveModelAlias(arg)
		if resolved == "" {
			labels := make([]string, len(store.AvailableModels))
			for i, m := range store.AvailableModels {
				labels[i] = m.Label
			}
			return c.Send("Unknown model. Available: " + strings.Join(labels, ", "))
		}
		if err := store.UpsertSettings(tid, &store.UserSettings{Model: resolved}); err != nil {
			return c.Send("Failed to update model.")
		}
		modelInfo := findModel(resolved)
		label := resolved
		if modelInfo != nil {
			label = modelInfo.Label
		}
		events.Bus.Emit(events.EventSettingChanged, map[string]any{"telegram_id": tid, "setting": "model", "value": resolved})
		return c.Send("Model set to: " + label)
	}

	settings := store.GetEffectiveSettings(tid, b.config)
	return c.Send("Select a model:", ModelPicker(settings.Model))
}

func (b *Bot) handleEffort(c tele.Context) error {
	tid := telegramID(c)
	arg := strings.ToLower(normalizeCommandArg(c.Message().Payload))

	if arg != "" {
		valid := false
		for _, level := range store.EffortLevels {
			if level == arg {
				valid = true
				break
			}
		}
		if !valid {
			settings := store.GetEffectiveSettings(tid, b.config)
			return c.Send(fmt.Sprintf("Current effort: %s\n\nSelect effort level:", settings.Effort), EffortPicker(settings.Effort))
		}
		if err := store.UpsertSettings(tid, &store.UserSettings{Effort: arg}); err != nil {
			return c.Send("Failed to update effort.")
		}
		events.Bus.Emit(events.EventSettingChanged, map[string]any{"telegram_id": tid, "setting": "effort", "value": arg})
		return c.Send("Effort set to: " + arg)
	}

	settings := store.GetEffectiveSettings(tid, b.config)
	return c.Send(fmt.Sprintf("Current effort: %s\n\nSelect effort level:", settings.Effort), EffortPicker(settings.Effort))
}

func (b *Bot) handleSettings(c tele.Context) error {
	tid := telegramID(c)
	settings := store.GetEffectiveSettings(tid, b.config)
	return c.Send(
		fmt.Sprintf("<b>Settings</b>\n%s\n\nTap a button to change:", store.FormatSettings(settings)),
		&tele.SendOptions{ParseMode: tele.ModeHTML},
		SettingsMenu(),
	)
}
