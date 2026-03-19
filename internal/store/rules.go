package store

import (
	"github.com/user/telegram-claude-bot/internal/rules"
)

type Rule = rules.Rule

var (
	AddGlobalRule    = rules.AddGlobalRule
	AddUserRule      = rules.AddUserRule
	RemoveGlobalRule = rules.RemoveGlobalRule
	RemoveUserRule   = rules.RemoveUserRule
	ToggleUserRule   = rules.ToggleUserRule
	ListGlobalRules  = rules.ListGlobalRules
	ListUserRules    = rules.ListUserRules
	GetActiveRules   = rules.GetActiveRules
	FormatRuleList   = rules.FormatRuleList
)
