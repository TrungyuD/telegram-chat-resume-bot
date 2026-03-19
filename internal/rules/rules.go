package rules

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/user/telegram-claude-bot/internal/platform/storage"
)

// Rule represents a global or user rule.
type Rule struct {
	Name      string `json:"name"`
	Content   string `json:"content"`
	IsActive  bool   `json:"is_active"`
	Priority  int    `json:"priority"`
	CreatedAt string `json:"created_at"`
}

func globalRulePath(name string) string {
	return filepath.Join(storage.DataDir, "rules", "global", storage.SafeFilename(name)+".json")
}

func userRulePath(telegramID, name string) string {
	return filepath.Join(storage.DataDir, "rules", "users", telegramID, storage.SafeFilename(name)+".json")
}

func AddGlobalRule(name, content string) (bool, error) {
	path := globalRulePath(name)
	unlock := storage.LockFile(path)
	defer unlock()
	if storage.FileExists(path) {
		return false, nil
	}
	rule := &Rule{
		Name:      name,
		Content:   content,
		IsActive:  true,
		CreatedAt: storage.NowUTC(),
	}
	return true, storage.WriteJSON(path, rule)
}

func AddUserRule(telegramID, name, content string) (bool, error) {
	path := userRulePath(telegramID, name)
	unlock := storage.LockFile(path)
	defer unlock()
	if storage.FileExists(path) {
		return false, nil
	}
	rule := &Rule{
		Name:      name,
		Content:   content,
		IsActive:  true,
		CreatedAt: storage.NowUTC(),
	}
	return true, storage.WriteJSON(path, rule)
}

func RemoveGlobalRule(name string) (bool, error) {
	path := globalRulePath(name)
	unlock := storage.LockFile(path)
	defer unlock()
	if !storage.FileExists(path) {
		return false, nil
	}
	return true, storage.DeleteFile(path)
}

func RemoveUserRule(telegramID, name string) (bool, error) {
	path := userRulePath(telegramID, name)
	unlock := storage.LockFile(path)
	defer unlock()
	return true, storage.DeleteFile(path)
}

func ToggleUserRule(telegramID, name string) (found, isActive bool, err error) {
	path := userRulePath(telegramID, name)
	unlock := storage.LockFile(path)
	defer unlock()
	if !storage.FileExists(path) {
		return false, false, nil
	}
	rule, err := storage.ReadJSON[Rule](path)
	if err != nil {
		return false, false, err
	}
	rule.IsActive = !rule.IsActive
	err = storage.WriteJSON(path, rule)
	return true, rule.IsActive, err
}

func ListGlobalRules() ([]*Rule, error) {
	dir := filepath.Join(storage.DataDir, "rules", "global")
	names, err := storage.ListJSONFiles(dir)
	if err != nil {
		return nil, err
	}
	list := make([]*Rule, 0, len(names))
	for _, name := range names {
		r, err := storage.ReadJSON[Rule](filepath.Join(dir, name+".json"))
		if err != nil {
			continue
		}
		list = append(list, &r)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].Priority > list[j].Priority
	})
	return list, nil
}

func ListUserRules(telegramID string) ([]*Rule, error) {
	dir := filepath.Join(storage.DataDir, "rules", "users", telegramID)
	names, err := storage.ListJSONFiles(dir)
	if err != nil {
		return nil, err
	}
	list := make([]*Rule, 0, len(names))
	for _, name := range names {
		r, err := storage.ReadJSON[Rule](filepath.Join(dir, name+".json"))
		if err != nil {
			continue
		}
		list = append(list, &r)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].Priority > list[j].Priority
	})
	return list, nil
}

func GetActiveRules(telegramID string) (global, user []*Rule, err error) {
	allGlobal, err := ListGlobalRules()
	if err != nil {
		return nil, nil, err
	}
	for _, r := range allGlobal {
		if r.IsActive {
			global = append(global, r)
		}
	}
	allUser, err := ListUserRules(telegramID)
	if err != nil {
		return nil, nil, err
	}
	for _, r := range allUser {
		if r.IsActive {
			user = append(user, r)
		}
	}
	return global, user, nil
}

func FormatRuleList(rules []*Rule) string {
	if len(rules) == 0 {
		return "No rules found."
	}
	var sb strings.Builder
	for _, r := range rules {
		status := "on"
		if !r.IsActive {
			status = "off"
		}
		fmt.Fprintf(&sb, "- **%s** [%s]: %s\n", r.Name, status, r.Content)
	}
	return sb.String()
}
