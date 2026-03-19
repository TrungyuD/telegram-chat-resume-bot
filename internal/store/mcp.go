package store

import (
	"github.com/TrungyuD/telegram-chat-resume-bot/internal/mcp"
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
