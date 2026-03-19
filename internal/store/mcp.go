package store

import (
	"github.com/user/telegram-claude-bot/internal/mcp"
)

type McpServer = mcp.McpServer

var (
	AddMcpServer         = mcp.AddMcpServer
	ParseMcpServerConfig = mcp.ParseMcpServerConfig
	RemoveMcpServer      = mcp.RemoveMcpServer
	ToggleMcpServer      = mcp.ToggleMcpServer
	ListMcpServers       = mcp.ListMcpServers
	ListActiveMcpServers = mcp.ListActiveMcpServers
	BuildMcpConfigs      = mcp.BuildMcpConfigs
	FormatServerList     = mcp.FormatServerList
)
