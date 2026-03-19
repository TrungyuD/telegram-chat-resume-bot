package store

import (
	"github.com/user/telegram-claude-bot/internal/config"
	"github.com/user/telegram-claude-bot/internal/settings"
)

// Re-export types for backward compatibility.
type UserSettings = settings.UserSettings
type EffectiveSettings = settings.EffectiveSettings
type ModelInfo = settings.ModelInfo

var (
	AvailableModels   = settings.AvailableModels
	EffortLevels      = settings.EffortLevels
	ThinkingModes     = settings.ThinkingModes
	GetSettings       = settings.GetSettings
	UpsertSettings    = settings.UpsertSettings
	ResolveModelAlias = settings.ResolveModelAlias
	FormatSettings    = settings.FormatSettings
)

// GetEffectiveSettings wraps settings.GetEffectiveSettings with config.GlobalConfig.
func GetEffectiveSettings(telegramID string, cfg *config.GlobalConfig) *settings.EffectiveSettings {
	return settings.GetEffectiveSettings(telegramID, cfg)
}
