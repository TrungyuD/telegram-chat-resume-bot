package store

import (
	"github.com/TrungyuD/telegram-chat-resume-bot/internal/memory"
)

type Memory = memory.Memory

var (
	SetMemory        = memory.SetMemory
	GetMemory        = memory.GetMemory
	ListMemory       = memory.ListMemory
	DeleteMemory     = memory.DeleteMemory
	ClearMemory      = memory.ClearMemory
	FormatMemoryList = memory.FormatMemoryList
	SearchMemory     = memory.SearchMemory
)
