package users

import (
	"os"
	"path/filepath"

	"github.com/user/telegram-claude-bot/internal/platform/storage"
)

// User represents a Telegram user profile.
type User struct {
	TelegramID       string `json:"telegram_id"`
	Username         string `json:"username"`
	DisplayName      string `json:"display_name"`
	Role             string `json:"role"`
	IsWhitelisted    bool   `json:"is_whitelisted"`
	WorkingDirectory string `json:"working_directory"`
	CreatedAt        string `json:"created_at"`
}

func userPath(telegramID string) string {
	return filepath.Join(storage.DataDir, "users", telegramID+".json")
}

func GetUser(telegramID string) (*User, error) {
	u, err := storage.ReadJSON[User](userPath(telegramID))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func CreateUser(user *User) error {
	unlock := storage.LockFile(userPath(user.TelegramID))
	defer unlock()
	if user.CreatedAt == "" {
		user.CreatedAt = storage.NowUTC()
	}
	return storage.WriteJSON(userPath(user.TelegramID), user)
}

func UpdateUser(user *User) error {
	unlock := storage.LockFile(userPath(user.TelegramID))
	defer unlock()
	return storage.WriteJSON(userPath(user.TelegramID), user)
}

func SetWhitelist(telegramID string, whitelisted bool) error {
	unlock := storage.LockFile(userPath(telegramID))
	defer unlock()
	u, err := storage.ReadJSON[User](userPath(telegramID))
	if err != nil {
		return err
	}
	u.IsWhitelisted = whitelisted
	return storage.WriteJSON(userPath(telegramID), u)
}

func SetWorkingDir(telegramID, dir string) error {
	unlock := storage.LockFile(userPath(telegramID))
	defer unlock()
	u, err := storage.ReadJSON[User](userPath(telegramID))
	if err != nil {
		return err
	}
	u.WorkingDirectory = dir
	return storage.WriteJSON(userPath(telegramID), u)
}

func ListAllUsers() ([]*User, error) {
	dir := filepath.Join(storage.DataDir, "users")
	names, err := storage.ListJSONFiles(dir)
	if err != nil {
		return nil, err
	}
	list := make([]*User, 0, len(names))
	for _, name := range names {
		u, err := storage.ReadJSON[User](filepath.Join(dir, name+".json"))
		if err != nil {
			continue
		}
		list = append(list, &u)
	}
	return list, nil
}

func DeleteUser(telegramID string) error {
	unlock := storage.LockFile(userPath(telegramID))
	defer unlock()
	return storage.DeleteFile(userPath(telegramID))
}
