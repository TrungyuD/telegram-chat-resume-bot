package store

import (
	"github.com/TrungyuD/telegram-chat-resume-bot/internal/sessions"
)

var ErrSessionNotFound = sessions.ErrSessionNotFound

var (
	GetSession             = sessions.GetSession
	GetActiveSession       = sessions.GetActiveSession
	SaveSession            = sessions.SaveSession
	GetSessionForDir       = sessions.GetSessionForDir
	ListSessions           = sessions.ListSessions
	SwitchSession          = sessions.SwitchSession
	DeactivateSession      = sessions.DeactivateSession
	DeleteSession          = sessions.DeleteSession
	GetSessionMessages     = sessions.GetSessionMessages
	GetSessionMessageCount = sessions.GetSessionMessageCount
	UpdateSessionLastUsed  = sessions.UpdateSessionLastUsed
	GetSessionPath         = sessions.GetSessionPath
)
