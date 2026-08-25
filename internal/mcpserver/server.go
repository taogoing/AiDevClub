package mcpserver

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"aidevclub/internal/platform"
)

const serverVersion = "0.1.0"

type Dependencies struct {
	Public  PublicDependencies
	Account AccountDependencies
}

func newMCPServer(deps Dependencies, actor Actor, cfg *platform.Config) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "aidevclub",
		Version: serverVersion,
	}, nil)
	RegisterPublicTools(server, deps.Public, cfg.PublicBaseURL)
	if actor.Authenticated {
		RegisterAccountTools(server, deps.Account, actor, cfg.PublicBaseURL)
	}
	return server
}
