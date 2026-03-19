package sessions

import (
	"errors"
	"path/filepath"

	"github.com/TrungyuD/telegram-chat-resume-bot/internal/platform/storage"
)

var ErrSessionNotFound = errors.New("session not found")

func sessionPath(telegramID, sessionID string) string {
	return filepath.Join(storage.DataDir, "sessions", telegramID, sessionID+".md")
}

// GetSessionPath returns the filesystem path for a session file.
func GetSessionPath(telegramID, sessionID string) string {
	return sessionPath(telegramID, sessionID)
}

func GetSession(telegramID, sessionID string) (*storage.SessionMeta, error) {
	path := sessionPath(telegramID, sessionID)
	meta, _, err := storage.ParseSessionMD(path)
	if err != nil {
		if !storage.FileExists(path) {
			return nil, ErrSessionNotFound
		}
		return nil, err
	}
	return meta, nil
}

func GetActiveSession(telegramID string) (*storage.SessionMeta, error) {
	dir := filepath.Join(storage.DataDir, "sessions", telegramID)
	names, err := storage.ListMDFiles(dir)
	if err != nil {
		return nil, err
	}
	for _, name := range names {
		path := filepath.Join(dir, name+".md")
		meta, _, err := storage.ParseSessionMD(path)
		if err != nil {
			continue
		}
		if meta.IsActive {
			return meta, nil
		}
	}
	return nil, nil
}

func SaveSession(meta *storage.SessionMeta) error {
	path := sessionPath(meta.TelegramID, meta.SessionID)
	if storage.FileExists(path) {
		return storage.UpdateSessionFrontmatter(path, meta)
	}
	return storage.WriteSessionMD(path, meta, nil)
}

func GetSessionForDir(telegramID, workingDir string) (*storage.SessionMeta, error) {
	dir := filepath.Join(storage.DataDir, "sessions", telegramID)
	names, err := storage.ListMDFiles(dir)
	if err != nil {
		return nil, err
	}
	var best *storage.SessionMeta
	for _, name := range names {
		path := filepath.Join(dir, name+".md")
		meta, _, err := storage.ParseSessionMD(path)
		if err != nil {
			continue
		}
		if meta.WorkingDir != workingDir {
			continue
		}
		if best == nil || sessionSortKey(meta) > sessionSortKey(best) {
			best = meta
		}
	}
	return best, nil
}

func ListSessions(telegramID string) ([]*storage.SessionMeta, error) {
	dir := filepath.Join(storage.DataDir, "sessions", telegramID)
	names, err := storage.ListMDFiles(dir)
	if err != nil {
		return nil, err
	}
	list := make([]*storage.SessionMeta, 0, len(names))
	for _, name := range names {
		path := filepath.Join(dir, name+".md")
		meta, _, err := storage.ParseSessionMD(path)
		if err != nil {
			continue
		}
		list = append(list, meta)
	}
	return list, nil
}

func SwitchSession(telegramID, sessionID string) error {
	dir := filepath.Join(storage.DataDir, "sessions", telegramID)
	// Lock the entire user session directory to prevent TOCTOU races
	unlock := storage.LockFile(dir)
	defer unlock()

	names, err := storage.ListMDFiles(dir)
	if err != nil {
		return err
	}

	type sessionUpdate struct {
		path string
		meta *storage.SessionMeta
	}

	updates := make([]sessionUpdate, 0, len(names))
	found := false
	for _, name := range names {
		path := filepath.Join(dir, name+".md")
		meta, _, err := storage.ParseSessionMD(path)
		if err != nil {
			continue
		}
		if meta.SessionID == sessionID {
			found = true
		}
		updates = append(updates, sessionUpdate{path: path, meta: meta})
	}
	if !found {
		return ErrSessionNotFound
	}

	for _, item := range updates {
		isActive := item.meta.SessionID == sessionID
		if item.meta.IsActive == isActive {
			continue
		}
		item.meta.IsActive = isActive
		if err := storage.UpdateSessionFrontmatter(item.path, item.meta); err != nil {
			return err
		}
	}
	return nil
}

func DeactivateSession(telegramID string) error {
	dir := filepath.Join(storage.DataDir, "sessions", telegramID)
	// Lock the entire user session directory to prevent TOCTOU races
	unlock := storage.LockFile(dir)
	defer unlock()

	names, err := storage.ListMDFiles(dir)
	if err != nil {
		return err
	}
	for _, name := range names {
		path := filepath.Join(dir, name+".md")
		meta, _, err := storage.ParseSessionMD(path)
		if err != nil {
			continue
		}
		if meta.IsActive {
			meta.IsActive = false
			if err := storage.UpdateSessionFrontmatter(path, meta); err != nil {
				return err
			}
		}
	}
	return nil
}

func DeleteSession(telegramID, sessionID string) error {
	path := sessionPath(telegramID, sessionID)
	unlock := storage.LockFile(path)
	defer unlock()
	return storage.DeleteFile(path)
}

func GetSessionMessages(telegramID, sessionID string) ([]storage.SessionMessage, error) {
	path := sessionPath(telegramID, sessionID)
	_, messages, err := storage.ParseSessionMD(path)
	if err != nil {
		return nil, err
	}
	return messages, nil
}

func GetSessionMessageCount(telegramID, sessionID string) (int, error) {
	messages, err := GetSessionMessages(telegramID, sessionID)
	if err != nil {
		return 0, err
	}
	return len(messages), nil
}

func UpdateSessionLastUsed(telegramID, sessionID string) error {
	path := sessionPath(telegramID, sessionID)
	meta, _, err := storage.ParseSessionMD(path)
	if err != nil {
		return err
	}
	meta.LastUsed = storage.NowUTC()
	return storage.UpdateSessionFrontmatter(path, meta)
}

func sessionSortKey(meta *storage.SessionMeta) string {
	if meta == nil {
		return ""
	}
	if meta.LastUsed != "" {
		return meta.LastUsed
	}
	return meta.CreatedAt
}
