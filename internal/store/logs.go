package store

import (
	"github.com/user/telegram-claude-bot/internal/logs"
)

type ActivityLog = logs.ActivityLog

var (
	AddLog   = logs.AddLog
	GetLogs  = logs.GetLogs
	GetStats = logs.GetStats
)
