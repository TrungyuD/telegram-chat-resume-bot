package logs

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/TrungyuD/telegram-chat-resume-bot/internal/platform/storage"
)

// ActivityLog represents a single log entry.
type ActivityLog struct {
	Type      string         `json:"type"`
	Level     string         `json:"level"`
	Message   string         `json:"message"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	CreatedAt string         `json:"created_at"`
}

func logPath(date string) string {
	return filepath.Join(storage.DataDir, "logs", date+".json")
}

func todayDate() string {
	return time.Now().UTC().Format("2006-01-02")
}

func AddLog(logType, level, message string, metadata map[string]any) error {
	date := todayDate()
	path := logPath(date)
	unlock := storage.LockFile(path)
	defer unlock()

	entry := ActivityLog{
		Type:      logType,
		Level:     level,
		Message:   message,
		Metadata:  metadata,
		CreatedAt: storage.NowUTC(),
	}

	var entries []ActivityLog
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &entries)
	}
	entries = append(entries, entry)
	return storage.WriteJSON(path, entries)
}

func GetLogs(date string, limit int) ([]*ActivityLog, error) {
	if date == "" {
		date = todayDate()
	}
	entries, err := storage.ReadJSON[[]ActivityLog](logPath(date))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	result := make([]*ActivityLog, 0, len(entries))
	for i := range entries {
		result = append(result, &entries[i])
	}
	if limit > 0 && len(result) > limit {
		result = result[len(result)-limit:]
	}
	return result, nil
}

func GetStats() (map[string]any, error) {
	entries, err := GetLogs(todayDate(), 0)
	if err != nil {
		return nil, err
	}
	errorCount := 0
	for _, l := range entries {
		if l.Level == "error" {
			errorCount++
		}
	}
	return map[string]any{
		"total_logs_today": len(entries),
		"error_count":      errorCount,
	}, nil
}
