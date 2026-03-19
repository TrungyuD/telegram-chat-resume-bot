package sessions

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/TrungyuD/telegram-chat-resume-bot/internal/platform/storage"
)

var ErrSessionNotFound = errors.New("session not found")

func sessionDir(telegramID string) string {
	return filepath.Join(storage.DataDir, "sessions", telegramID)
}

func sessionPath(telegramID, sessionID string) string {
	return filepath.Join(sessionDir(telegramID), sessionID+".md")
}

func activeSessionCachePath(telegramID string) string {
	return filepath.Join(sessionDir(telegramID), "active_session.txt")
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

// writeActiveCache writes the active session ID to a cache file.
func writeActiveCache(telegramID, sessionID string) {
	path := activeSessionCachePath(telegramID)
	_ = storage.EnsureDir(filepath.Dir(path))
	_ = os.WriteFile(path, []byte(sessionID), 0o644)
}

// clearActiveCache removes the active session cache file.
func clearActiveCache(telegramID string) {
	_ = os.Remove(activeSessionCachePath(telegramID))
}

func GetActiveSession(telegramID string) (*storage.SessionMeta, error) {
	// Try cached active session first
	if data, err := os.ReadFile(activeSessionCachePath(telegramID)); err == nil {
		cachedID := strings.TrimSpace(string(data))
		if cachedID != "" {
			meta, err := GetSession(telegramID, cachedID)
			if err == nil && meta != nil && meta.IsActive {
				return meta, nil
			}
			// Cache is stale, clear it
			clearActiveCache(telegramID)
		}
	}

	// Fallback: scan all session files
	dir := sessionDir(telegramID)
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
			// Update cache for next time
			writeActiveCache(telegramID, meta.SessionID)
			return meta, nil
		}
	}
	return nil, nil
}

func SaveSession(meta *storage.SessionMeta) error {
	path := sessionPath(meta.TelegramID, meta.SessionID)
	var err error
	if storage.FileExists(path) {
		err = storage.UpdateSessionFrontmatter(path, meta)
	} else {
		err = storage.WriteSessionMD(path, meta, nil)
	}
	if err != nil {
		return err
	}
	// Update active session cache
	if meta.IsActive {
		writeActiveCache(meta.TelegramID, meta.SessionID)
	}
	return nil
}

func GetSessionForDir(telegramID, workingDir string) (*storage.SessionMeta, error) {
	dir := sessionDir(telegramID)
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
	dir := sessionDir(telegramID)
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
	dir := sessionDir(telegramID)
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

	// Update active session cache
	writeActiveCache(telegramID, sessionID)
	return nil
}

func DeactivateSession(telegramID string) error {
	dir := sessionDir(telegramID)
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

	// Clear active session cache
	clearActiveCache(telegramID)
	return nil
}

func DeleteSession(telegramID, sessionID string) error {
	path := sessionPath(telegramID, sessionID)
	unlock := storage.LockFile(path)
	defer unlock()

	// Clear cache if deleting the active session
	if data, err := os.ReadFile(activeSessionCachePath(telegramID)); err == nil {
		if strings.TrimSpace(string(data)) == sessionID {
			clearActiveCache(telegramID)
		}
	}

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
