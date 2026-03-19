package mcp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/TrungyuD/telegram-chat-resume-bot/internal/platform/storage"
)

// McpServer represents an MCP server configuration.
type McpServer struct {
	Name      string            `json:"name"`
	Type      string            `json:"type"`
	Command   string            `json:"command,omitempty"`
	Args      []string          `json:"args,omitempty"`
	URL       string            `json:"url,omitempty"`
	Env       map[string]string `json:"env,omitempty"`
	IsActive  bool              `json:"is_active"`
	CreatedAt string            `json:"created_at"`
}

type stdioMcpConfig struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env"`
}

type remoteMcpConfig struct {
	URL string `json:"url"`
}

func mcpPath(name string) string {
	return filepath.Join(storage.DataDir, "mcp", storage.SafeFilename(name)+".json")
}

func AddMcpServer(server *McpServer) error {
	if err := validateMcpServer(server); err != nil {
		return err
	}

	path := mcpPath(server.Name)
	unlock := storage.LockFile(path)
	defer unlock()
	if server.CreatedAt == "" {
		server.CreatedAt = storage.NowUTC()
	}
	return storage.WriteJSON(path, server)
}

func ParseMcpServerConfig(name, serverType, rawConfig string) (*McpServer, error) {
	server := &McpServer{
		Name:      strings.TrimSpace(name),
		Type:      strings.ToLower(strings.TrimSpace(serverType)),
		IsActive:  true,
		CreatedAt: storage.NowUTC(),
	}
	if server.Name == "" {
		return nil, fmt.Errorf("server name is required")
	}

	switch server.Type {
	case "stdio":
		var cfg stdioMcpConfig
		if err := decodeStrictJSON(rawConfig, &cfg); err != nil {
			return nil, fmt.Errorf("invalid stdio config: %w", err)
		}
		server.Command = strings.TrimSpace(cfg.Command)
		server.Args = cfg.Args
		if len(cfg.Env) > 0 {
			server.Env = cfg.Env
		}
	case "http", "sse":
		var cfg remoteMcpConfig
		if err := decodeStrictJSON(rawConfig, &cfg); err != nil {
			return nil, fmt.Errorf("invalid %s config: %w", server.Type, err)
		}
		server.URL = strings.TrimSpace(cfg.URL)
	default:
		return nil, fmt.Errorf("unsupported MCP server type: %s", serverType)
	}

	if err := validateMcpServer(server); err != nil {
		return nil, err
	}
	return server, nil
}

func RemoveMcpServer(name string) (bool, error) {
	path := mcpPath(name)
	unlock := storage.LockFile(path)
	defer unlock()
	if !storage.FileExists(path) {
		return false, nil
	}
	return true, storage.DeleteFile(path)
}

func ToggleMcpServer(name string) (found, isActive bool, err error) {
	path := mcpPath(name)
	unlock := storage.LockFile(path)
	defer unlock()
	if !storage.FileExists(path) {
		return false, false, nil
	}
	server, err := storage.ReadJSON[McpServer](path)
	if err != nil {
		return false, false, err
	}
	server.IsActive = !server.IsActive
	err = storage.WriteJSON(path, server)
	return true, server.IsActive, err
}

func ListMcpServers() ([]*McpServer, error) {
	dir := filepath.Join(storage.DataDir, "mcp")
	names, err := storage.ListJSONFiles(dir)
	if err != nil {
		return nil, err
	}
	servers := make([]*McpServer, 0, len(names))
	for _, name := range names {
		s, err := storage.ReadJSON[McpServer](filepath.Join(dir, name+".json"))
		if err != nil {
			continue
		}
		servers = append(servers, &s)
	}
	return servers, nil
}

func ListActiveMcpServers() ([]*McpServer, error) {
	all, err := ListMcpServers()
	if err != nil {
		return nil, err
	}
	var active []*McpServer
	for _, s := range all {
		if s.IsActive {
			active = append(active, s)
		}
	}
	return active, nil
}

func BuildMcpConfigs() (map[string]any, error) {
	servers, err := ListActiveMcpServers()
	if err != nil {
		return nil, err
	}
	configs := make(map[string]any)
	for _, s := range servers {
		switch s.Type {
		case "stdio":
			entry := map[string]any{
				"command": s.Command,
				"args":    s.Args,
			}
			if len(s.Env) > 0 {
				entry["env"] = s.Env
			}
			configs[s.Name] = entry
		case "sse", "http":
			configs[s.Name] = map[string]any{
				"url": s.URL,
			}
		}
	}
	return configs, nil
}

func FormatServerList(servers []*McpServer) string {
	if len(servers) == 0 {
		return "No MCP servers configured."
	}
	var sb strings.Builder
	for _, s := range servers {
		status := "on"
		if !s.IsActive {
			status = "off"
		}
		detail := ""
		if s.Type == "stdio" {
			detail = fmt.Sprintf("%s %s", s.Command, strings.Join(s.Args, " "))
		} else {
			detail = s.URL
		}
		fmt.Fprintf(&sb, "- **%s** [%s/%s]: %s\n", s.Name, s.Type, status, detail)
	}
	return sb.String()
}

func decodeStrictJSON(raw string, target any) error {
	dec := json.NewDecoder(strings.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(target); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return fmt.Errorf("unexpected trailing data")
		}
		return err
	}
	return nil
}

func validateMcpServer(server *McpServer) error {
	if server == nil {
		return fmt.Errorf("server is required")
	}
	server.Name = strings.TrimSpace(server.Name)
	server.Type = strings.ToLower(strings.TrimSpace(server.Type))
	if server.Name == "" {
		return fmt.Errorf("server name is required")
	}

	switch server.Type {
	case "stdio":
		server.Command = strings.TrimSpace(server.Command)
		if server.Command == "" {
			return fmt.Errorf("stdio config requires command")
		}
		for i, arg := range server.Args {
			server.Args[i] = strings.TrimSpace(arg)
		}
		if len(server.Env) == 0 {
			server.Env = nil
		}
		server.URL = ""
	case "http", "sse":
		server.URL = strings.TrimSpace(server.URL)
		if server.URL == "" {
			return fmt.Errorf("%s config requires url", server.Type)
		}
		parsed, err := url.ParseRequestURI(server.URL)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return fmt.Errorf("%s config requires a valid url", server.Type)
		}
		server.Command = ""
		server.Args = nil
		server.Env = nil
	default:
		return fmt.Errorf("unsupported MCP server type: %s", server.Type)
	}
	return nil
}
