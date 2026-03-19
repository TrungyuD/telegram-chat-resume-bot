package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/TrungyuD/telegram-chat-resume-bot/internal/platform/storage"
)

// GlobalConfig holds all application configuration.
type GlobalConfig struct {
	TelegramBotToken   string
	AdminTelegramIDs   []string
	AllowedWorkingDirs []string
	WebPort            int
	DefaultModel       string
	DefaultEffort      string
	DefaultThinking    string
	MaxConcurrent      int
	TimeoutMs          int
	ContextMessages    int
	RateLimitPerMinute int
	AdminAPIKey        string
	CompactThreshold   int
	CompactKeepRecent  int
	CompactEnabled     bool
}

func configPath() string {
	return filepath.Join(storage.DataDir, "config.json")
}

func getEnvInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func getEnvBool(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}

func getEnvList(key string) []string {
	v := os.Getenv(key)
	if v == "" {
		return nil
	}
	var result []string
	for _, s := range strings.Split(v, ",") {
		s = strings.TrimSpace(s)
		if s != "" {
			result = append(result, s)
		}
	}
	return result
}

func getEnvIntWithAliases(keys []string, def int) int {
	for _, key := range keys {
		v := os.Getenv(key)
		if v == "" {
			continue
		}
		n, err := strconv.Atoi(v)
		if err == nil {
			return n
		}
	}
	return def
}

func splitTrimmed(s string) []string {
	var result []string
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func firstConfigValue(m map[string]string, keys ...string) string {
	for _, key := range keys {
		if v, ok := m[key]; ok && v != "" {
			return v
		}
	}
	return ""
}

// LoadGlobalConfig loads configuration from env vars and data/config.json overrides.
func LoadGlobalConfig() *GlobalConfig {
	cfg := &GlobalConfig{
		TelegramBotToken:   os.Getenv("TELEGRAM_BOT_TOKEN"),
		AdminTelegramIDs:   getEnvList("ADMIN_TELEGRAM_IDS"),
		AllowedWorkingDirs: getEnvList("ALLOWED_WORKING_DIRS"),
		WebPort:            getEnvInt("WEB_PORT", 3000),
		DefaultModel:       os.Getenv("CLAUDE_DEFAULT_MODEL"),
		DefaultEffort:      os.Getenv("CLAUDE_DEFAULT_EFFORT"),
		DefaultThinking:    os.Getenv("CLAUDE_DEFAULT_THINKING"),
		MaxConcurrent:      getEnvIntWithAliases([]string{"MAX_CONCURRENT_CLI_PROCESSES", "MAX_CONCURRENT"}, 3),
		TimeoutMs:          getEnvIntWithAliases([]string{"CLAUDE_CLI_TIMEOUT_MS", "TIMEOUT_MS"}, 360000),
		ContextMessages:    getEnvIntWithAliases([]string{"CLAUDE_CONTEXT_MESSAGES", "CONTEXT_MESSAGES"}, 10),
		RateLimitPerMinute: getEnvIntWithAliases([]string{"RATE_LIMIT_REQUESTS_PER_MINUTE", "RATE_LIMIT_PER_MINUTE"}, 10),
		AdminAPIKey:        os.Getenv("ADMIN_API_KEY"),
		CompactThreshold:   getEnvInt("COMPACT_THRESHOLD", 20),
		CompactKeepRecent:  getEnvInt("COMPACT_KEEP_RECENT", 6),
		CompactEnabled:     getEnvBool("COMPACT_ENABLED", true),
	}

	// Apply defaults if empty
	if cfg.DefaultModel == "" {
		cfg.DefaultModel = "claude-sonnet-4-6"
	}
	if cfg.DefaultEffort == "" {
		cfg.DefaultEffort = "medium"
	}
	if cfg.DefaultThinking == "" {
		cfg.DefaultThinking = "adaptive"
	}

	// Override with data/config.json values
	fileConf, err := GetAllConfig()
	if err == nil {
		applyFileOverrides(cfg, fileConf)
	}

	return cfg
}

func applyFileOverrides(cfg *GlobalConfig, fileConf map[string]string) {
	if v := firstConfigValue(fileConf, "TELEGRAM_BOT_TOKEN", "telegram_bot_token"); v != "" {
		cfg.TelegramBotToken = v
	}
	if v := firstConfigValue(fileConf, "ADMIN_TELEGRAM_IDS", "admin_telegram_ids"); v != "" {
		cfg.AdminTelegramIDs = splitTrimmed(v)
	}
	if v := firstConfigValue(fileConf, "ALLOWED_WORKING_DIRS", "allowed_working_dirs"); v != "" {
		cfg.AllowedWorkingDirs = splitTrimmed(v)
	}
	if v := firstConfigValue(fileConf, "WEB_PORT", "web_port"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.WebPort = n
		}
	}
	if v := firstConfigValue(fileConf, "CLAUDE_DEFAULT_MODEL", "claude_default_model"); v != "" {
		cfg.DefaultModel = v
	}
	if v := firstConfigValue(fileConf, "CLAUDE_DEFAULT_EFFORT", "claude_default_effort"); v != "" {
		cfg.DefaultEffort = v
	}
	if v := firstConfigValue(fileConf, "CLAUDE_DEFAULT_THINKING", "claude_default_thinking"); v != "" {
		cfg.DefaultThinking = v
	}
	if v := firstConfigValue(fileConf, "MAX_CONCURRENT_CLI_PROCESSES", "MAX_CONCURRENT", "max_concurrent_cli_processes", "max_concurrent"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.MaxConcurrent = n
		}
	}
	if v := firstConfigValue(fileConf, "CLAUDE_CLI_TIMEOUT_MS", "TIMEOUT_MS", "claude_cli_timeout_ms", "timeout_ms"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.TimeoutMs = n
		}
	}
	if v := firstConfigValue(fileConf, "CLAUDE_CONTEXT_MESSAGES", "CONTEXT_MESSAGES", "claude_context_messages", "context_messages"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.ContextMessages = n
		}
	}
	if v := firstConfigValue(fileConf, "RATE_LIMIT_REQUESTS_PER_MINUTE", "RATE_LIMIT_PER_MINUTE", "rate_limit_requests_per_minute", "rate_limit_per_minute"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.RateLimitPerMinute = n
		}
	}
	if v := firstConfigValue(fileConf, "ADMIN_API_KEY", "admin_api_key"); v != "" {
		cfg.AdminAPIKey = v
	}
	if v := firstConfigValue(fileConf, "COMPACT_THRESHOLD", "compact_threshold"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.CompactThreshold = n
		}
	}
	if v := firstConfigValue(fileConf, "COMPACT_KEEP_RECENT", "compact_keep_recent"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.CompactKeepRecent = n
		}
	}
	if v := firstConfigValue(fileConf, "COMPACT_ENABLED", "compact_enabled"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			cfg.CompactEnabled = b
		}
	}
}

// GetConfig reads a single key from data/config.json.
func GetConfig(key string) string {
	m, err := GetAllConfig()
	if err != nil {
		return ""
	}
	return m[key]
}

// SetConfig writes a key-value pair to data/config.json.
func SetConfig(key, value string) error {
	path := configPath()
	unlock := storage.LockFile(path)
	defer unlock()

	m, err := GetAllConfig()
	if err != nil {
		m = make(map[string]string)
	}
	m[key] = value
	return storage.WriteJSON(path, m)
}

// GetAllConfig reads all key-value pairs from data/config.json.
func GetAllConfig() (map[string]string, error) {
	data, err := os.ReadFile(configPath())
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]string), nil
		}
		return nil, err
	}
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}
