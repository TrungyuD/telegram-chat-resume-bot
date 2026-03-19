package store

import (
	"github.com/TrungyuD/telegram-chat-resume-bot/internal/users"
)

// User re-exported for backward compatibility.
type User = users.User

var (
	GetUser       = users.GetUser
	CreateUser    = users.CreateUser
	UpdateUser    = users.UpdateUser
	SetWhitelist  = users.SetWhitelist
	SetWorkingDir = users.SetWorkingDir
	ListAllUsers  = users.ListAllUsers
	DeleteUser    = users.DeleteUser
)
