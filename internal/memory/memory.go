package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/user/telegram-claude-bot/internal/platform/storage"
)

// Memory represents a user memory entry.
type Memory struct {
	Key       string `json:"key"`
	Value     string `json:"value"`
	Category  string `json:"category"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func memoryPath(telegramID, key string) string {
	return filepath.Join(storage.DataDir, "memory", telegramID, storage.SafeFilename(key)+".json")
}

func SetMemory(telegramID, key, value, category string) error {
	path := memoryPath(telegramID, key)
	unlock := storage.LockFile(path)
	defer unlock()

	var m Memory
	existing, err := storage.ReadJSON[Memory](path)
	if err == nil {
		m = existing
	} else {
		m = Memory{
			Key:       key,
			Category:  category,
			CreatedAt: storage.NowUTC(),
		}
	}
	m.Value = value
	if category != "" {
		m.Category = category
	}
	m.UpdatedAt = storage.NowUTC()
	return storage.WriteJSON(path, m)
}

func GetMemory(telegramID, key string) (*Memory, error) {
	m, err := storage.ReadJSON[Memory](memoryPath(telegramID, key))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

func ListMemory(telegramID string) ([]*Memory, error) {
	dir := filepath.Join(storage.DataDir, "memory", telegramID)
	names, err := storage.ListJSONFiles(dir)
	if err != nil {
		return nil, err
	}
	memories := make([]*Memory, 0, len(names))
	for _, name := range names {
		m, err := storage.ReadJSON[Memory](filepath.Join(dir, name+".json"))
		if err != nil {
			continue
		}
		memories = append(memories, &m)
	}
	return memories, nil
}

func DeleteMemory(telegramID, key string) (bool, error) {
	path := memoryPath(telegramID, key)
	if !storage.FileExists(path) {
		return false, nil
	}
	unlock := storage.LockFile(path)
	defer unlock()
	return true, storage.DeleteFile(path)
}

func ClearMemory(telegramID string) (int, error) {
	dir := filepath.Join(storage.DataDir, "memory", telegramID)
	names, err := storage.ListJSONFiles(dir)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, name := range names {
		path := filepath.Join(dir, name+".json")
		unlock := storage.LockFile(path)
		if err := storage.DeleteFile(path); err == nil {
			count++
		}
		unlock()
	}
	return count, nil
}

func FormatMemoryList(memories []*Memory) string {
	if len(memories) == 0 {
		return "No memories found."
	}
	var sb strings.Builder
	for _, m := range memories {
		fmt.Fprintf(&sb, "- **%s**: %s\n", m.Key, m.Value)
	}
	return sb.String()
}

func SearchMemory(telegramID, query string) ([]*Memory, error) {
	all, err := ListMemory(telegramID)
	if err != nil {
		return nil, err
	}
	q := strings.ToLower(query)
	var results []*Memory
	for _, m := range all {
		if strings.Contains(strings.ToLower(m.Key), q) ||
			strings.Contains(strings.ToLower(m.Value), q) {
			results = append(results, m)
		}
	}
	return results, nil
}
