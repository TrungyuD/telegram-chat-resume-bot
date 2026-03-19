// Package store - config backward compatibility shim.
// New code should import "internal/config" directly.
package store

import (
	"github.com/user/telegram-claude-bot/internal/config"
)

// GlobalConfig re-exported for backward compatibility.
type GlobalConfig = config.GlobalConfig

// Re-export config functions for backward compatibility.
var (
	LoadGlobalConfig = config.LoadGlobalConfig
	GetConfig        = config.GetConfig
	SetConfig        = config.SetConfig
	GetAllConfig     = config.GetAllConfig
)
