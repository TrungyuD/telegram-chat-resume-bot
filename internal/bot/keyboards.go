package bot

import (
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/TrungyuD/telegram-chat-resume-bot/internal/store"
	tele "gopkg.in/telebot.v4"
)

// ModelPicker creates an inline keyboard for model selection.
func ModelPicker(currentModel string) *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{}
	rows := make([]tele.Row, 0, len(store.AvailableModels))
	for _, m := range store.AvailableModels {
		label := m.Label
		if m.ID == currentModel {
			label = "✓ " + label
		}
		rows = append(rows, menu.Row(menu.Data(label, "model_"+m.ID, "model:"+m.ID)))
	}
	menu.Inline(rows...)
	return menu
}

// EffortPicker creates an inline keyboard for effort level selection.
func EffortPicker(currentEffort string) *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{}
	btns := make([]tele.Btn, 0, len(store.EffortLevels))
	titler := cases.Title(language.English)
	for _, e := range store.EffortLevels {
		label := titler.String(e)
		if e == currentEffort {
			label = "✓ " + label
		}
		btns = append(btns, menu.Data(label, "effort_"+e, "effort:"+e))
	}
	menu.Inline(menu.Row(btns...))
	return menu
}

// SettingsMenu creates the main settings inline keyboard.
func SettingsMenu() *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{}
	menu.Inline(
		menu.Row(
			menu.Data("Model", "s_model", "settings:model"),
			menu.Data("Effort", "s_effort", "settings:effort"),
		),
		menu.Row(
			menu.Data("Rules", "s_rules", "settings:rules"),
			menu.Data("Memory", "s_memory", "settings:memory"),
			menu.Data("Sessions", "s_sessions", "settings:sessions"),
		),
	)
	return menu
}

// ConfirmDialog creates a yes/no confirmation keyboard.
func ConfirmDialog(action string) *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{}
	menu.Inline(menu.Row(
		menu.Data("Yes", "cf_yes", "confirm:"+action+":yes"),
		menu.Data("No", "cf_no", "confirm:"+action+":no"),
	))
	return menu
}

// SessionList creates an inline keyboard listing sessions.
func SessionList(sessions []*store.SessionMeta) *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{}
	limit := 5
	if len(sessions) < limit {
		limit = len(sessions)
	}
	rows := make([]tele.Row, 0, limit)
	for i := 0; i < limit; i++ {
		session := sessions[i]
		label := session.Title
		if label == "" {
			label = session.SessionID
			if len(label) > 12 {
				label = label[:12] + "..."
			}
		}
		if session.IsActive {
			label = "✓ " + label
		}
		rows = append(rows, menu.Row(menu.Data(label, "sess_"+session.SessionID, "session:"+session.SessionID)))
	}
	menu.Inline(rows...)
	return menu
}
