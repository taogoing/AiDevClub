package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"aidevclub/internal/model"
	"aidevclub/internal/service"
)

const testPublicBaseURL = "https://club.example.test"

type fakeSearchReader struct {
	response *service.SearchResponse
	err      error
	queries  []service.SearchQuery
}

func (f *fakeSearchReader) Search(_ context.Context, query service.SearchQuery) (*service.SearchResponse, error) {
	f.queries = append(f.queries, query)
	if f.err != nil {
		return nil, f.err
	}
	if f.response == nil {
		return &service.SearchResponse{
			Articles:   []service.SearchResult{},
			Skills:     []service.SearchResult{},
			McpServers: []service.SearchResult{},
			Page:       query.Page,
			PageSize:   query.PageSize,
		}, nil
	}
	return f.response, nil
}

type fakeArticleReader struct {
	detail    *service.ArticleDetail
	readErr   error
	list      *service.ArticleListResult
	listErr   error
	readCalls int
	listCalls int
	readID    uint
	readUser  uint
	listQuery service.ListQuery
}

func (f *fakeArticleReader) Read(_ context.Context, userID, articleID uint) (*service.ArticleDetail, error) {
	f.readCalls++
	f.readUser = userID
	f.readID = articleID
	if f.readErr != nil {
		return nil, f.readErr
	}
	return f.detail, nil
}

func (f *fakeArticleReader) List(_ context.Context, query service.ListQuery) (*service.ArticleListResult, error) {
	f.listCalls++
	f.listQuery = query
	if f.listErr != nil {
		return nil, f.listErr
	}
	if f.list == nil {
		return &service.ArticleListResult{List: []service.ArticleSummary{}, Page: query.Page, PageSize: query.PageSize}, nil
	}
	return f.list, nil
}

type fakeSkillReader struct {
	detail    *service.SkillDetail
	readErr   error
	list      *service.SkillListResult
	listErr   error
	readCalls int
	listCalls int
	readID    uint
	readUser  uint
	listQuery service.SkillListQuery
}

func (f *fakeSkillReader) Read(_ context.Context, userID, skillID uint) (*service.SkillDetail, error) {
	f.readCalls++
	f.readUser = userID
	f.readID = skillID
	if f.readErr != nil {
		return nil, f.readErr
	}
	return f.detail, nil
}

func (f *fakeSkillReader) List(_ context.Context, query service.SkillListQuery) (*service.SkillListResult, error) {
	f.listCalls++
	f.listQuery = query
	if f.listErr != nil {
		return nil, f.listErr
	}
	if f.list == nil {
		return &service.SkillListResult{List: []service.SkillSummary{}, Page: query.Page, PageSize: query.PageSize}, nil
	}
	return f.list, nil
}

type fakeMCPServerReader struct {
	detail    *service.McpServerDetail
	readErr   error
	list      *service.McpServerListResult
	listErr   error
	readCalls int
	listCalls int
	readID    uint
	readUser  uint
	listQuery service.McpServerListQuery
}

func (f *fakeMCPServerReader) Read(_ context.Context, userID, serverID uint) (*service.McpServerDetail, error) {
	f.readCalls++
	f.readUser = userID
	f.readID = serverID
	if f.readErr != nil {
		return nil, f.readErr
	}
	return f.detail, nil
}

func (f *fakeMCPServerReader) List(_ context.Context, query service.McpServerListQuery) (*service.McpServerListResult, error) {
	f.listCalls++
	f.listQuery = query
	if f.listErr != nil {
		return nil, f.listErr
	}
	if f.list == nil {
		return &service.McpServerListResult{List: []service.McpServerSummary{}, Page: query.Page, PageSize: query.PageSize}, nil
	}
	return f.list, nil
}

type fakeRankingReader struct {
	articles []service.ArticleSummary
	skills   []service.SkillSummary
	servers  []service.McpServerSummary
	err      error

	articleCalls int
	skillCalls   int
	serverCalls  int
}

func (f *fakeRankingReader) ListArticleHot(context.Context, int, int) ([]service.ArticleSummary, error) {
	f.articleCalls++
	if f.err != nil {
		return nil, f.err
	}
	return f.articles, nil
}

func (f *fakeRankingReader) ListSkillHot(context.Context, int, int) ([]service.SkillSummary, error) {
	f.skillCalls++
	if f.err != nil {
		return nil, f.err
	}
	return f.skills, nil
}

func (f *fakeRankingReader) ListMcpServerHot(context.Context, int, int) ([]service.McpServerSummary, error) {
	f.serverCalls++
	if f.err != nil {
		return nil, f.err
	}
	return f.servers, nil
}

type fakeCategoryReader struct {
	items   []model.Category
	err     error
	keyword string
	limit   int
	calls   int
}

func (f *fakeCategoryReader) ListForMCP(_ context.Context, keyword string, limit int) ([]model.Category, error) {
	f.calls++
	f.keyword = keyword
	f.limit = limit
	if f.err != nil {
		return nil, f.err
	}
	return f.items, nil
}

type fakeTagReader struct {
	items   []model.Tag
	err     error
	keyword string
	limit   int
	calls   int
}

func (f *fakeTagReader) ListForMCP(_ context.Context, keyword string, limit int) ([]model.Tag, error) {
	f.calls++
	f.keyword = keyword
	f.limit = limit
	if f.err != nil {
		return nil, f.err
	}
	return f.items, nil
}

func publicTestDependencies() PublicDependencies {
	return PublicDependencies{
		Search:     &fakeSearchReader{},
		Articles:   &fakeArticleReader{},
		Skills:     &fakeSkillReader{},
		MCPServers: &fakeMCPServerReader{},
		Ranking:    &fakeRankingReader{},
		Categories: &fakeCategoryReader{},
		Tags:       &fakeTagReader{},
	}
}

func newTestServer(deps PublicDependencies) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "aidevclub-test", Version: "test"}, nil)
	RegisterPublicTools(server, deps, testPublicBaseURL)
	return server
}

func callToolResult(t *testing.T, server *mcp.Server, name string, arguments map[string]any) *mcp.CallToolResult {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("connect server: %v", err)
	}
	defer serverSession.Close()
	client := mcp.NewClient(&mcp.Implementation{Name: "aidevclub-test-client", Version: "test"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("connect client: %v", err)
	}
	defer clientSession.Close()

	result, err := clientSession.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: arguments})
	if err != nil {
		t.Fatalf("call %s: %v", name, err)
	}
	return result
}

func callTool[T any](t *testing.T, server *mcp.Server, name string, arguments map[string]any) T {
	t.Helper()
	result := callToolResult(t, server, name, arguments)
	if result.IsError {
		t.Fatalf("call %s returned tool error: %s", name, toolText(result))
	}
	payload, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatalf("marshal %s structured content: %v", name, err)
	}
	var output T
	if err := json.Unmarshal(payload, &output); err != nil {
		t.Fatalf("decode %s structured content %s: %v", name, payload, err)
	}
	return output
}

func toolText(result *mcp.CallToolResult) string {
	if result == nil || len(result.Content) == 0 {
		return ""
	}
	text, _ := result.Content[0].(*mcp.TextContent)
	if text == nil {
		return ""
	}
	return text.Text
}

func assertToolErrorCode(t *testing.T, result *mcp.CallToolResult, code string) {
	t.Helper()
	if !result.IsError {
		t.Fatalf("IsError = false, want true; structured=%#v", result.StructuredContent)
	}
	text := toolText(result)
	if !strings.Contains(text, fmt.Sprintf(`"code":%q`, code)) {
		t.Fatalf("error text = %q, want stable code %q", text, code)
	}
}

func listPublicTools(t *testing.T, server *mcp.Server) []*mcp.Tool {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer serverSession.Close()
	clientSession, err := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "test"}, nil).Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer clientSession.Close()
	listed, err := clientSession.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	return listed.Tools
}

func toolInputSchema(t *testing.T, tools []*mcp.Tool, name string) map[string]any {
	t.Helper()

	for _, tool := range tools {
		if tool.Name != name {
			continue
		}
		payload, err := json.Marshal(tool.InputSchema)
		if err != nil {
			t.Fatalf("marshal %s input schema: %v", name, err)
		}
		var schema map[string]any
		if err := json.Unmarshal(payload, &schema); err != nil {
			t.Fatalf("decode %s input schema: %v", name, err)
		}
		return schema
	}
	t.Fatalf("tool %q not listed", name)
	return nil
}

func assertSchemaProperty(t *testing.T, schema map[string]any, property string, keywords map[string]any) {
	t.Helper()

	var visit func(any) bool
	visit = func(value any) bool {
		switch node := value.(type) {
		case map[string]any:
			if properties, ok := node["properties"].(map[string]any); ok {
				if candidate, ok := properties[property].(map[string]any); ok {
					matches := true
					for keyword, want := range keywords {
						if !reflect.DeepEqual(candidate[keyword], want) {
							matches = false
							break
						}
					}
					if matches {
						return true
					}
				}
			}
			for _, child := range node {
				if visit(child) {
					return true
				}
			}
		case []any:
			for _, child := range node {
				if visit(child) {
					return true
				}
			}
		}
		return false
	}
	if !visit(schema) {
		payload, _ := json.Marshal(schema)
		t.Fatalf("schema property %q has no occurrence with keywords %#v: %s", property, keywords, payload)
	}
}

func TestSearchContentRegistersExactlySixPublicTools(t *testing.T) {
	tools := listPublicTools(t, newTestServer(publicTestDependencies()))
	want := []string{"browse_content", "get_article", "get_mcp_server", "get_skill", "list_taxonomy", "search_content"}
	if len(tools) != len(want) {
		t.Fatalf("tool count = %d, want %d", len(tools), len(want))
	}
	got := make([]string, 0, len(tools))
	for _, tool := range tools {
		got = append(got, tool.Name)
		if tool.InputSchema == nil || tool.OutputSchema == nil {
			t.Fatalf("tool %q is missing an explicit SDK schema", tool.Name)
		}
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("tool names = %v, want %v", got, want)
		}
	}
}

func TestPublicToolInputSchemasAdvertiseConstraintsAndDefaults(t *testing.T) {
	tools := listPublicTools(t, newTestServer(publicTestDependencies()))

	search := toolInputSchema(t, tools, "search_content")
	assertSchemaProperty(t, search, "content_type", map[string]any{
		"enum": []any{"all", "article", "skill", "mcp_server"}, "default": "all",
	})
	assertSchemaProperty(t, search, "sort", map[string]any{"enum": []any{"relevance", "latest"}})
	assertSchemaProperty(t, search, "sort", map[string]any{"default": "relevance"})
	assertSchemaProperty(t, search, "sort", map[string]any{"default": "latest"})
	assertSchemaProperty(t, search, "tag_id", map[string]any{"minimum": float64(1)})
	assertSchemaProperty(t, search, "category_id", map[string]any{"minimum": float64(1)})
	assertSchemaProperty(t, search, "page", map[string]any{"minimum": float64(1), "default": float64(1)})
	assertSchemaProperty(t, search, "page_size", map[string]any{"minimum": float64(1), "maximum": float64(20)})
	assertSchemaProperty(t, search, "page_size", map[string]any{"maximum": float64(10), "default": float64(5)})
	assertSchemaProperty(t, search, "page_size", map[string]any{"maximum": float64(20), "default": float64(10)})

	browse := toolInputSchema(t, tools, "browse_content")
	assertSchemaProperty(t, browse, "content_type", map[string]any{
		"enum": []any{"all", "article", "skill", "mcp_server"}, "default": "all",
	})
	assertSchemaProperty(t, browse, "sort", map[string]any{
		"enum": []any{"latest", "hot"}, "default": "latest",
	})
	assertSchemaProperty(t, browse, "page", map[string]any{"minimum": float64(1), "default": float64(1)})
	assertSchemaProperty(t, browse, "page_size", map[string]any{"minimum": float64(1), "maximum": float64(20)})
	assertSchemaProperty(t, browse, "page_size", map[string]any{"maximum": float64(10), "default": float64(5)})
	assertSchemaProperty(t, browse, "page_size", map[string]any{"maximum": float64(20), "default": float64(10)})

	for _, name := range []string{"get_article", "get_skill", "get_mcp_server"} {
		schema := toolInputSchema(t, tools, name)
		assertSchemaProperty(t, schema, "id", map[string]any{"minimum": float64(1)})
		assertSchemaProperty(t, schema, "content_offset", map[string]any{"minimum": float64(0), "default": float64(0)})
		assertSchemaProperty(t, schema, "content_limit", map[string]any{
			"minimum": float64(1), "maximum": float64(50000), "default": float64(30000),
		})
	}

	taxonomy := toolInputSchema(t, tools, "list_taxonomy")
	assertSchemaProperty(t, taxonomy, "kind", map[string]any{
		"enum": []any{"all", "categories", "tags"}, "default": "all",
	})
	assertSchemaProperty(t, taxonomy, "limit", map[string]any{
		"minimum": float64(1), "maximum": float64(100), "default": float64(50),
	})
}

func TestSearchContentRejectsInvalidContentTypeAndSort(t *testing.T) {
	tests := []struct {
		name string
		args map[string]any
	}{
		{name: "content type", args: map[string]any{"query": "Go", "content_type": "video"}},
		{name: "sort", args: map[string]any{"query": "Go", "content_type": "article", "sort": "hot"}},
		{name: "relevance needs query", args: map[string]any{"tag_id": 1, "content_type": "article", "sort": "relevance"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			deps := publicTestDependencies()
			result := callToolResult(t, newTestServer(deps), "search_content", test.args)
			assertToolErrorCode(t, result, errorCodeInvalidArgument)
			if calls := len(deps.Search.(*fakeSearchReader).queries); calls != 0 {
				t.Fatalf("search calls = %d, want 0 for invalid input", calls)
			}
		})
	}
}

func TestSearchContentValidatesFilterAndCategoryCompatibility(t *testing.T) {
	tests := []map[string]any{
		{"content_type": "article"},
		{"category_id": 3, "content_type": "all", "sort": "latest"},
		{"category_id": 3, "content_type": "skill", "sort": "latest"},
		{"category_id": 3, "content_type": "mcp_server", "sort": "latest"},
	}
	for _, arguments := range tests {
		result := callToolResult(t, newTestServer(publicTestDependencies()), "search_content", arguments)
		assertToolErrorCode(t, result, errorCodeInvalidArgument)
	}
}

func TestSearchContentAllUsesTypedPlainTextSections(t *testing.T) {
	search := &fakeSearchReader{response: &service.SearchResponse{
		Articles:   []service.SearchResult{{ID: 1, Type: "article", Title: "Go <MCP>", Summary: "plain article", Views: 11}},
		Skills:     []service.SearchResult{{ID: 2, Type: "skill", Title: "Go Skill", Summary: "plain skill", Views: 12}},
		McpServers: []service.SearchResult{{ID: 3, Type: "mcp_server", Title: "Go Server", Summary: "plain server", Views: 13}},
		Items:      []service.SearchResult{{ID: 99, Type: "article", Title: "legacy item must not leak"}},
		Counts:     map[string]int64{"article": 100, "skill": 200, "mcp_server": 300},
		Total:      600, Page: 1, PageSize: 5,
	}}
	deps := publicTestDependencies()
	deps.Search = search
	server := newTestServer(deps)
	result := callToolResult(t, server, "search_content", map[string]any{
		"query": "Go", "content_type": "all", "sort": "relevance",
	})
	if result.IsError {
		t.Fatalf("tool error: %s", toolText(result))
	}
	payload, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	var output searchContentOutput
	if err := json.Unmarshal(payload, &output); err != nil {
		t.Fatal(err)
	}
	if len(output.Articles) != 1 || len(output.Skills) != 1 || len(output.MCPServers) != 1 {
		t.Fatalf("typed sections = articles:%d skills:%d mcp:%d, want 1 each", len(output.Articles), len(output.Skills), len(output.MCPServers))
	}
	if output.Articles[0].Title != "Go <MCP>" || strings.Contains(output.Articles[0].Title, "<mark>") {
		t.Fatalf("article title = %q, want plain non-highlighted text", output.Articles[0].Title)
	}
	if output.Articles[0].URL != "https://club.example.test/articles/1" || output.Skills[0].URL != "https://club.example.test/skills/2" || output.MCPServers[0].URL != "https://club.example.test/mcps/3" {
		t.Fatalf("page URLs = %q %q %q", output.Articles[0].URL, output.Skills[0].URL, output.MCPServers[0].URL)
	}
	if len(search.queries) != 1 || search.queries[0].Highlight {
		t.Fatalf("search queries = %#v, want one with Highlight false", search.queries)
	}
	text := string(payload)
	if strings.Contains(text, "legacy item must not leak") || strings.Contains(text, `"items"`) || strings.Contains(text, `"counts"`) {
		t.Fatalf("MCP output leaked legacy SearchResponse fields: %s", payload)
	}
	if summary := toolText(result); summary == "" || strings.HasPrefix(strings.TrimSpace(summary), "{") {
		t.Fatalf("text content = %q, want concise human-readable summary", summary)
	}
}

func TestSearchContentLatestUsesDomainListReader(t *testing.T) {
	publishedAt := time.Date(2026, 8, 25, 9, 10, 11, 0, time.UTC)
	articles := &fakeArticleReader{list: &service.ArticleListResult{
		List:  []service.ArticleSummary{{ID: 7, Title: "Latest", Summary: "from list", PublishedAt: &publishedAt}},
		Total: 1, Page: 2, PageSize: 4,
	}}
	search := &fakeSearchReader{}
	deps := publicTestDependencies()
	deps.Articles = articles
	deps.Search = search
	output := callTool[searchContentOutput](t, newTestServer(deps), "search_content", map[string]any{
		"tag_id": 8, "content_type": "article", "sort": "latest", "page": 2, "page_size": 4,
	})
	if len(search.queries) != 0 || articles.listCalls != 1 {
		t.Fatalf("search calls = %d, article list calls = %d; want 0/1", len(search.queries), articles.listCalls)
	}
	if articles.listQuery.Sort != "latest" || articles.listQuery.TagID == nil || *articles.listQuery.TagID != 8 {
		t.Fatalf("article list query = %#v", articles.listQuery)
	}
	if output.Total != 1 || output.Page != 2 || output.PageSize != 4 || len(output.Articles) != 1 {
		t.Fatalf("output = %#v", output)
	}
	if output.Articles[0].PublishedAt != "2026-08-25T09:10:11Z" {
		t.Fatalf("published_at = %q, want RFC3339", output.Articles[0].PublishedAt)
	}
}

func TestSearchContentReturnsEmptyArrays(t *testing.T) {
	search := &fakeSearchReader{response: &service.SearchResponse{
		Articles: []service.SearchResult{}, Skills: []service.SearchResult{}, McpServers: []service.SearchResult{},
		Page: 1, PageSize: 5,
	}}
	deps := publicTestDependencies()
	deps.Search = search
	result := callToolResult(t, newTestServer(deps), "search_content", map[string]any{"query": "none", "content_type": "all"})
	payload, _ := json.Marshal(result.StructuredContent)
	for _, field := range []string{`"articles":[]`, `"skills":[]`, `"mcp_servers":[]`} {
		if !strings.Contains(string(payload), field) {
			t.Fatalf("structured content %s does not contain %s", payload, field)
		}
	}
}

func TestBrowseContentRejectsInvalidCombinations(t *testing.T) {
	tests := []map[string]any{
		{"content_type": "video", "sort": "latest"},
		{"content_type": "article", "sort": "downloads"},
		{"content_type": "all", "sort": "downloads"},
		{"content_type": "skill", "sort": "relevance"},
	}
	for _, arguments := range tests {
		result := callToolResult(t, newTestServer(publicTestDependencies()), "browse_content", arguments)
		assertToolErrorCode(t, result, errorCodeInvalidArgument)
	}
}

func TestBrowseContentAllHotUsesRankingSections(t *testing.T) {
	ranking := &fakeRankingReader{
		articles: []service.ArticleSummary{{ID: 10, Title: "Hot article", Tags: []service.TagBrief{}}},
		skills:   []service.SkillSummary{{ID: 20, Name: "Hot skill", Tags: []service.TagBrief{}}},
		servers:  []service.McpServerSummary{{ID: 30, Name: "Hot MCP", Tags: []service.TagBrief{}}},
	}
	deps := publicTestDependencies()
	deps.Ranking = ranking
	output := callTool[browseContentOutput](t, newTestServer(deps), "browse_content", map[string]any{
		"content_type": "all", "sort": "hot", "page": 1, "page_size": 5,
	})
	if ranking.articleCalls != 1 || ranking.skillCalls != 1 || ranking.serverCalls != 1 {
		t.Fatalf("ranking calls = %d/%d/%d, want 1/1/1", ranking.articleCalls, ranking.skillCalls, ranking.serverCalls)
	}
	if len(output.Articles) != 1 || len(output.Skills) != 1 || len(output.MCPServers) != 1 {
		t.Fatalf("output sections = %#v", output)
	}
	if output.Articles[0].URL != testPublicBaseURL+"/articles/10" || output.Skills[0].URL != testPublicBaseURL+"/skills/20" || output.MCPServers[0].URL != testPublicBaseURL+"/mcps/30" {
		t.Fatalf("non-absolute or incorrect URLs: %#v", output)
	}
}

func TestGetArticleUsesReadAndReturnsUnicodeWindow(t *testing.T) {
	article := &fakeArticleReader{detail: &service.ArticleDetail{
		ArticleSummary: service.ArticleSummary{ID: 9, Title: "Unicode", Tags: []service.TagBrief{}},
		Content:        strings.Repeat("界", 20),
	}}
	deps := publicTestDependencies()
	deps.Articles = article
	output := callTool[getArticleOutput](t, newTestServer(deps), "get_article", map[string]any{
		"id": 9, "content_offset": 5, "content_limit": 7,
	})
	if output.Content != "界界界界界界界" || !output.HasMore || output.NextOffset != 12 {
		t.Fatalf("window = content:%q has_more:%v next:%d", output.Content, output.HasMore, output.NextOffset)
	}
	if output.URL != testPublicBaseURL+"/articles/9" {
		t.Fatalf("url = %q", output.URL)
	}
	if article.readCalls != 1 || article.readID != 9 || article.readUser != 0 {
		t.Fatalf("Read calls/id/user = %d/%d/%d", article.readCalls, article.readID, article.readUser)
	}
}

func TestGetArticleRejectsInvalidWindowBeforeRead(t *testing.T) {
	tests := []map[string]any{
		{"id": 1, "content_offset": -1},
		{"id": 1, "content_limit": -1},
		{"id": 1, "content_limit": 50001},
		{"id": 0},
	}
	for _, arguments := range tests {
		article := &fakeArticleReader{}
		deps := publicTestDependencies()
		deps.Articles = article
		result := callToolResult(t, newTestServer(deps), "get_article", arguments)
		assertToolErrorCode(t, result, errorCodeInvalidArgument)
		if article.readCalls != 0 {
			t.Fatalf("Read calls = %d, want 0 for %#v", article.readCalls, arguments)
		}
	}
}

func TestGetArticleMapsReaderFailureWithoutLeakingDatabaseDetails(t *testing.T) {
	article := &fakeArticleReader{readErr: errors.New("SELECT * FROM articles: password=secret")}
	deps := publicTestDependencies()
	deps.Articles = article
	result := callToolResult(t, newTestServer(deps), "get_article", map[string]any{"id": 9})
	assertToolErrorCode(t, result, errorCodeInternal)
	if strings.Contains(toolText(result), "SELECT") || strings.Contains(toolText(result), "secret") {
		t.Fatalf("reader details leaked in %q", toolText(result))
	}
}

func TestGetSkillReturnsPersistedSkillMDWindow(t *testing.T) {
	skill := &fakeSkillReader{detail: &service.SkillDetail{
		SkillSummary: service.SkillSummary{ID: 12, Name: "Agent Skill", Tags: []service.TagBrief{}},
		SkillMD:      "甲乙丙丁戊己庚辛",
	}}
	deps := publicTestDependencies()
	deps.Skills = skill
	result := callToolResult(t, newTestServer(deps), "get_skill", map[string]any{
		"id": 12, "content_offset": 2, "content_limit": 3,
	})
	if result.IsError {
		t.Fatalf("tool error: %s", toolText(result))
	}
	payload, _ := json.Marshal(result.StructuredContent)
	var output getSkillOutput
	if err := json.Unmarshal(payload, &output); err != nil {
		t.Fatal(err)
	}
	if output.SkillMD != "丙丁戊" || !output.HasMore || output.NextOffset != 5 {
		t.Fatalf("skill_md window = %#v", output)
	}
	if output.URL != testPublicBaseURL+"/skills/12" || strings.Contains(string(payload), "zip_url") || strings.Contains(string(payload), "downloads") {
		t.Fatalf("resource output contains a legacy field or has wrong URL: %s", payload)
	}
	if skill.readCalls != 1 || skill.readUser != 0 {
		t.Fatalf("Read calls/user = %d/%d", skill.readCalls, skill.readUser)
	}
}

func TestGetMCPServerReturnsInstallationsAndReadmeWindow(t *testing.T) {
	reader := &fakeMCPServerReader{detail: &service.McpServerDetail{
		McpServerSummary: service.McpServerSummary{ID: 15, Name: "Toolbox", Tags: []service.TagBrief{}},
		Installations:    []service.McpInstallation{{Client: "cursor", Command: "npx -y toolbox"}},
		Readme:           "零一二三四五六七八九",
	}}
	deps := publicTestDependencies()
	deps.MCPServers = reader
	result := callToolResult(t, newTestServer(deps), "get_mcp_server", map[string]any{
		"id": 15, "content_offset": 3, "content_limit": 4,
	})
	if result.IsError {
		t.Fatalf("tool error: %s", toolText(result))
	}
	payload, _ := json.Marshal(result.StructuredContent)
	var output getMCPServerOutput
	if err := json.Unmarshal(payload, &output); err != nil {
		t.Fatal(err)
	}
	if output.Readme != "三四五六" || !output.HasMore || output.NextOffset != 7 {
		t.Fatalf("readme window = %#v", output)
	}
	if len(output.Installations) != 1 || output.Installations[0].Client != "cursor" {
		t.Fatalf("installations = %#v", output.Installations)
	}
	if strings.Contains(string(payload), "tools_json") || strings.Contains(string(payload), "zip_url") || strings.Contains(string(payload), "downloads") {
		t.Fatalf("legacy MCP fields leaked: %s", payload)
	}
	if output.URL != testPublicBaseURL+"/mcps/15" {
		t.Fatalf("resource metadata = %#v", output)
	}
}

func TestListTaxonomyValidatesKind(t *testing.T) {
	result := callToolResult(t, newTestServer(publicTestDependencies()), "list_taxonomy", map[string]any{"kind": "authors"})
	assertToolErrorCode(t, result, errorCodeInvalidArgument)
}

func TestListTaxonomyReturnsFilteredKindsAndEmptyArrays(t *testing.T) {
	categories := &fakeCategoryReader{items: []model.Category{{ID: 1, Name: "Go", Slug: "go"}}}
	tags := &fakeTagReader{items: []model.Tag{{ID: 2, Name: "MCP", Description: "Protocol", Enabled: true}}}
	deps := publicTestDependencies()
	deps.Categories = categories
	deps.Tags = tags
	all := callTool[listTaxonomyOutput](t, newTestServer(deps), "list_taxonomy", map[string]any{
		"kind": "all", "keyword": "go", "limit": 7,
	})
	if len(all.Categories) != 1 || len(all.Tags) != 1 {
		t.Fatalf("taxonomy = %#v", all)
	}
	if categories.keyword != "go" || tags.keyword != "go" || categories.limit != 7 || tags.limit != 7 {
		t.Fatalf("taxonomy reader args = categories:%q/%d tags:%q/%d", categories.keyword, categories.limit, tags.keyword, tags.limit)
	}

	deps = publicTestDependencies()
	deps.Categories = categories
	deps.Tags = tags
	result := callToolResult(t, newTestServer(deps), "list_taxonomy", map[string]any{"kind": "categories"})
	payload, _ := json.Marshal(result.StructuredContent)
	if !strings.Contains(string(payload), `"tags":[]`) || !strings.Contains(string(payload), `"categories":[`) {
		t.Fatalf("taxonomy empty-array contract not preserved: %s", payload)
	}
}
