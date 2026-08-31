package mcpserver

import (
	"context"
	"fmt"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"aidevclub/internal/model"
	"aidevclub/internal/service"
)

type SearchReader interface {
	Search(context.Context, service.SearchQuery) (*service.SearchResponse, error)
}

type ArticleReader interface {
	Read(context.Context, uint, uint) (*service.ArticleDetail, error)
	List(context.Context, service.ListQuery) (*service.ArticleListResult, error)
}

type SkillReader interface {
	Read(context.Context, uint, uint) (*service.SkillDetail, error)
	List(context.Context, service.SkillListQuery) (*service.SkillListResult, error)
}

type MCPServerReader interface {
	Read(context.Context, uint, uint) (*service.McpServerDetail, error)
	List(context.Context, service.McpServerListQuery) (*service.McpServerListResult, error)
}

type RankingReader interface {
	ListArticleHot(context.Context, int, int) ([]service.ArticleSummary, error)
	ListSkillHot(context.Context, int, int) ([]service.SkillSummary, error)
	ListMcpServerHot(context.Context, int, int) ([]service.McpServerSummary, error)
}

type TagReader interface {
	ListForMCP(context.Context, string, int) ([]model.Tag, error)
}

type ProfileReader interface {
	Get(context.Context, uint) (*model.User, error)
}

type OwnedArticleReader interface {
	ListOwned(context.Context, uint, string, int, int) (*service.ArticleListResult, error)
}

type OwnedSkillReader interface {
	ListOwned(context.Context, uint, string, int, int) (*service.SkillListResult, error)
}

type OwnedMCPServerReader interface {
	ListOwned(context.Context, uint, string, int, int) (*service.McpServerListResult, error)
}

type NotificationReader interface {
	List(context.Context, uint, string, bool, int, int) (*service.NotificationListResult, error)
}

type PublicDependencies struct {
	Search     SearchReader
	Articles   ArticleReader
	Skills     SkillReader
	MCPServers MCPServerReader
	Ranking    RankingReader
	Tags       TagReader
}

type AccountDependencies struct {
	Profile       ProfileReader
	Articles      OwnedArticleReader
	Skills        OwnedSkillReader
	MCPServers    OwnedMCPServerReader
	Notifications NotificationReader
}

func mustInputSchema[T any]() *jsonschema.Schema {
	schema, err := jsonschema.For[T](nil)
	if err != nil {
		panic(fmt.Sprintf("infer MCP input schema: %v", err))
	}
	return schema
}

func RegisterPublicTools(server *mcp.Server, deps PublicDependencies, publicBaseURL string) {
	server.AddReceivingMiddleware(stablePublicToolSchemaErrors)
	annotations := &mcp.ToolAnnotations{ReadOnlyHint: true, IdempotentHint: true}
	mcp.AddTool(server, &mcp.Tool{
		Name: "search_content", Description: "Search published AIDevClub content.", Annotations: annotations,
		InputSchema: searchContentInputSchema(),
	}, searchContent(deps, publicBaseURL))
	mcp.AddTool(server, &mcp.Tool{
		Name: "browse_content", Description: "Browse latest or hot AIDevClub content.", Annotations: annotations,
		InputSchema: browseContentInputSchema(),
	}, browseContent(deps, publicBaseURL))
	mcp.AddTool(server, &mcp.Tool{
		Name: "get_article", Description: "Read a published article without changing its view count.", Annotations: annotations,
		InputSchema: getArticleInputSchema(),
	}, getArticle(deps.Articles, publicBaseURL))
	mcp.AddTool(server, &mcp.Tool{
		Name: "get_skill", Description: "Read a published Skill and its SKILL.md.", Annotations: annotations,
		InputSchema: getSkillInputSchema(),
	}, getSkill(deps.Skills, publicBaseURL))
	mcp.AddTool(server, &mcp.Tool{
		Name: "get_mcp_server", Description: "Read a published MCP Server definition.", Annotations: annotations,
		InputSchema: getMCPServerInputSchema(),
	}, getMCPServer(deps.MCPServers, publicBaseURL))
	mcp.AddTool(server, &mcp.Tool{
		Name: "list_taxonomy", Description: "List enabled tags.", Annotations: annotations,
		InputSchema: listTaxonomyInputSchema(),
	}, listTaxonomy(deps, publicBaseURL))
}
