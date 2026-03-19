package costs

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/TrungyuD/telegram-chat-resume-bot/internal/platform/storage"
)

// CostRecord represents a single cost entry.
type CostRecord struct {
	SessionID    string  `json:"session_id"`
	CostUSD      float64 `json:"cost_usd"`
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	Model        string  `json:"model"`
	CreatedAt    string  `json:"created_at"`
}

// costMonthPath returns the path for a user's cost file for a given month.
func costMonthPath(telegramID, month string) string {
	return filepath.Join(storage.DataDir, "costs", telegramID+"_"+month+".json")
}

// currentMonth returns the current month in YYYY-MM format.
func currentMonth() string {
	return time.Now().UTC().Format("2006-01")
}

// legacyCostPath returns the old single-file path for migration.
func legacyCostPath(telegramID string) string {
	return filepath.Join(storage.DataDir, "costs", telegramID+".json")
}

func AddCostRecord(telegramID string, record *CostRecord) error {
	if record.CreatedAt == "" {
		record.CreatedAt = storage.NowUTC()
	}

	// Determine month from record timestamp
	month := currentMonth()
	if len(record.CreatedAt) >= 7 {
		month = record.CreatedAt[:7]
	}

	path := costMonthPath(telegramID, month)
	unlock := storage.LockFile(path)
	defer unlock()

	var records []CostRecord
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &records)
	}
	records = append(records, *record)
	return storage.WriteJSON(path, records)
}

// readUserCostFiles reads all cost files (legacy + monthly) for a user.
func readUserCostFiles(telegramID string) []CostRecord {
	var all []CostRecord

	// Read legacy file if it exists
	if records, err := storage.ReadJSON[[]CostRecord](legacyCostPath(telegramID)); err == nil {
		all = append(all, records...)
	}

	// Read monthly files
	dir := filepath.Join(storage.DataDir, "costs")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return all
	}
	prefix := telegramID + "_"
	for _, e := range entries {
		name := e.Name()
		if !strings.HasPrefix(name, prefix) || !strings.HasSuffix(name, ".json") {
			continue
		}
		records, err := storage.ReadJSON[[]CostRecord](filepath.Join(dir, name))
		if err != nil {
			continue
		}
		all = append(all, records...)
	}
	return all
}

func GetUserTotalCost(telegramID string) float64 {
	records := readUserCostFiles(telegramID)
	var total float64
	for _, r := range records {
		total += r.CostUSD
	}
	return total
}

func GetUserCostToday(telegramID string) float64 {
	// Only need to read current month's file + legacy
	today := time.Now().UTC().Format("2006-01-02")
	month := today[:7]

	var total float64

	// Check legacy file
	if records, err := storage.ReadJSON[[]CostRecord](legacyCostPath(telegramID)); err == nil {
		for _, r := range records {
			if strings.HasPrefix(r.CreatedAt, today) {
				total += r.CostUSD
			}
		}
	}

	// Check current month file
	if records, err := storage.ReadJSON[[]CostRecord](costMonthPath(telegramID, month)); err == nil {
		for _, r := range records {
			if strings.HasPrefix(r.CreatedAt, today) {
				total += r.CostUSD
			}
		}
	}

	return total
}

func GetAllCostStats() map[string]float64 {
	dir := filepath.Join(storage.DataDir, "costs")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	stats := make(map[string]float64)
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".json") {
			continue
		}

		records, err := storage.ReadJSON[[]CostRecord](filepath.Join(dir, name))
		if err != nil {
			continue
		}

		// Extract telegram ID: either "tid.json" (legacy) or "tid_YYYY-MM.json" (monthly)
		base := strings.TrimSuffix(name, ".json")
		tid := base
		if idx := strings.LastIndex(base, "_"); idx > 0 {
			// Check if suffix looks like a month (YYYY-MM)
			suffix := base[idx+1:]
			if len(suffix) == 7 && suffix[4] == '-' {
				tid = base[:idx]
			}
		}

		for _, r := range records {
			stats[tid] += r.CostUSD
		}
	}
	return stats
}
