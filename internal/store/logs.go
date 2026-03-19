package store

import (
	"github.com/TrungyuD/telegram-chat-resume-bot/internal/logs"
)

type ActivityLog = logs.ActivityLog

var (
	AddLog   = logs.AddLog
	GetLogs  = logs.GetLogs
	GetStats = logs.GetStats
)
