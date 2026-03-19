package settings

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/TrungyuD/telegram-chat-resume-bot/internal/config"
	"github.com/TrungyuD/telegram-chat-resume-bot/internal/platform/storage"
)

// UserSettings holds per-user AI settings.
type UserSettings struct {
	Model        string   `json:"model,omitempty"`
	Effort       string   `json:"effort,omitempty"`
	Thinking     string   `json:"thinking,omitempty"`
	MaxTurns     *int     `json:"max_turns,omitempty"`
	MaxBudgetUSD *float64 `json:"max_budget_usd,omitempty"`
	UpdatedAt    string   `json:"updated_at,omitempty"`
}

// EffectiveSettings holds the resolved settings for a user.
type EffectiveSettings struct {
	Model        string
	Effort       string
	Thinking     string
	MaxTurns     *int
	MaxBudgetUSD *float64
}

// ModelInfo describes an available AI model.
type ModelInfo struct {
	ID    string
	Label string
}

var AvailableModels = []ModelInfo{
	{ID: "claude-sonnet-4-6", Label: "Sonnet 4.6"},
	{ID: "claude-opus-4-6", Label: "Opus 4.6"},
	{ID: "claude-haiku-4-5", Label: "Haiku 4.5"},
}

var EffortLevels = []string{"low", "medium", "high"}
var ThinkingModes = []string{"off", "adaptive", "on"}

func settingsPath(telegramID string) string {
	return filepath.Join(storage.DataDir, "settings", telegramID+".json")
}

func GetSettings(telegramID string) (*UserSettings, error) {
	s, err := storage.ReadJSON[UserSettings](settingsPath(telegramID))
	if err != nil {
		if os.IsNotExist(err) {
			return &UserSettings{}, nil
		}
		return nil, err
	}
	return &s, nil
}

func UpsertSettings(telegramID string, patch *UserSettings) error {
	if patch == nil {
		return fmt.Errorf("settings patch is nil")
	}

	path := settingsPath(telegramID)
	unlock := storage.LockFile(path)
	defer unlock()

	current := &UserSettings{}
	existing, err := storage.ReadJSON[UserSettings](path)
	if err == nil {
		current = &existing
	} else if !os.IsNotExist(err) {
		return err
	}

	merged := mergeUserSettings(current, patch)
	merged.UpdatedAt = storage.NowUTC()
	return storage.WriteJSON(path, merged)
}

func mergeUserSettings(current, patch *UserSettings) *UserSettings {
	if current == nil {
		current = &UserSettings{}
	}
	if patch == nil {
		return current
	}

	merged := *current
	if patch.Model != "" {
		merged.Model = patch.Model
	}
	if patch.Effort != "" {
		merged.Effort = patch.Effort
	}
	if patch.Thinking != "" {
		merged.Thinking = patch.Thinking
	}
	if patch.MaxTurns != nil {
		merged.MaxTurns = patch.MaxTurns
	}
	if patch.MaxBudgetUSD != nil {
		merged.MaxBudgetUSD = patch.MaxBudgetUSD
	}

	return &merged
}

func ResolveModelAlias(input string) string {
	aliases := map[string]string{
		"sonnet": "claude-sonnet-4-6",
		"opus":   "claude-opus-4-6",
		"haiku":  "claude-haiku-4-5",
	}
	if id, ok := aliases[input]; ok {
		return id
	}
	for _, m := range AvailableModels {
		if m.ID == input {
			return input
		}
	}
	return ""
}

func GetEffectiveSettings(telegramID string, cfg *config.GlobalConfig) *EffectiveSettings {
	s, _ := GetSettings(telegramID)
	if s == nil {
		s = &UserSettings{}
	}

	effective := &EffectiveSettings{
		Model:        cfg.DefaultModel,
		Effort:       normalizeEffort(cfg.DefaultEffort),
		Thinking:     cfg.DefaultThinking,
		MaxTurns:     s.MaxTurns,
		MaxBudgetUSD: s.MaxBudgetUSD,
	}
	if s.Model != "" {
		effective.Model = s.Model
	}
	if s.Effort != "" {
		effective.Effort = normalizeEffort(s.Effort)
	}
	if s.Thinking != "" {
		effective.Thinking = s.Thinking
	}
	return effective
}

func FormatSettings(s *EffectiveSettings) string {
	maxTurns := "default"
	if s.MaxTurns != nil {
		maxTurns = fmt.Sprintf("%d", *s.MaxTurns)
	}
	maxBudget := "default"
	if s.MaxBudgetUSD != nil {
		maxBudget = fmt.Sprintf("$%.2f", *s.MaxBudgetUSD)
	}
	return fmt.Sprintf(
		"Model: %s\nEffort: %s\nMax Turns: %s\nMax Budget: %s",
		s.Model, s.Effort, maxTurns, maxBudget,
	)
}

func normalizeEffort(value string) string {
	switch value {
	case "max":
		return "high"
	case "low", "medium", "high":
		return value
	default:
		return "medium"
	}
}
