package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"aidevclub/internal/service"
)

type searchContentInput struct {
	Query       string `json:"query,omitempty" jsonschema:"Search keywords."`
	ContentType string `json:"content_type,omitempty" jsonschema:"Content type: all, article, skill, or mcp_server."`
	TagID       *uint  `json:"tag_id,omitempty" jsonschema:"Enabled tag ID filter."`
	CategoryID  *uint  `json:"category_id,omitempty" jsonschema:"Article category ID filter."`
	Sort        string `json:"sort,omitempty" jsonschema:"Sort mode: relevance or latest."`
	Page        int    `json:"page,omitempty" jsonschema:"Page number, starting at 1."`
	PageSize    int    `json:"page_size,omitempty" jsonschema:"Results per type."`
}

type contentSummaryOutput struct {
	ID          uint         `json:"id"`
	Type        string       `json:"type"`
	Title       string       `json:"title"`
	Summary     string       `json:"summary"`
	URL         string       `json:"url"`
	Author      AuthorOutput `json:"author"`
	Tags        []TagOutput  `json:"tags"`
	Views       int          `json:"views"`
	Downloads   int          `json:"downloads,omitempty"`
	PublishedAt string       `json:"published_at,omitempty"`
}

type searchContentOutput struct {
	ContentType string                 `json:"content_type"`
	Sort        string                 `json:"sort"`
	Articles    []contentSummaryOutput `json:"articles"`
	Skills      []contentSummaryOutput `json:"skills"`
	MCPServers  []contentSummaryOutput `json:"mcp_servers"`
	Total       int64                  `json:"total"`
	PageInfo
}

type browseContentInput struct {
	ContentType string `json:"content_type,omitempty" jsonschema:"Content type: all, article, skill, or mcp_server."`
	Sort        string `json:"sort,omitempty" jsonschema:"Sort mode: latest, hot, or downloads."`
	Page        int    `json:"page,omitempty" jsonschema:"Page number, starting at 1."`
	PageSize    int    `json:"page_size,omitempty" jsonschema:"Results per type."`
}

type browseContentOutput struct {
	ContentType string                 `json:"content_type"`
	Sort        string                 `json:"sort"`
	Articles    []contentSummaryOutput `json:"articles"`
	Skills      []contentSummaryOutput `json:"skills"`
	MCPServers  []contentSummaryOutput `json:"mcp_servers"`
	Total       int64                  `json:"total"`
	PageInfo
}

func searchContentInputSchema() *jsonschema.Schema {
	schema := mustInputSchema[searchContentInput]()
	schema.Properties["content_type"].Enum = []any{"all", "article", "skill", "mcp_server"}
	schema.Properties["content_type"].Default = json.RawMessage(`"all"`)
	schema.Properties["sort"].Enum = []any{"relevance", "latest"}
	schema.Properties["tag_id"].Minimum = jsonschema.Ptr(float64(1))
	schema.Properties["category_id"].Minimum = jsonschema.Ptr(float64(1))
	setPagedContentSchema(schema)
	schema.AllOf = append(schema.AllOf, &jsonschema.Schema{
		If: &jsonschema.Schema{
			Required: []string{"query"},
			Properties: map[string]*jsonschema.Schema{
				"query": {Pattern: `\S`},
			},
		},
		Then: &jsonschema.Schema{Properties: map[string]*jsonschema.Schema{
			"sort": {Default: json.RawMessage(`"relevance"`)},
		}},
		Else: &jsonschema.Schema{Properties: map[string]*jsonschema.Schema{
			"sort": {Default: json.RawMessage(`"latest"`)},
		}},
	})
	return schema
}

func browseContentInputSchema() *jsonschema.Schema {
	schema := mustInputSchema[browseContentInput]()
	schema.Properties["content_type"].Enum = []any{"all", "article", "skill", "mcp_server"}
	schema.Properties["content_type"].Default = json.RawMessage(`"all"`)
	schema.Properties["sort"].Enum = []any{"latest", "hot", "downloads"}
	schema.Properties["sort"].Default = json.RawMessage(`"latest"`)
	setPagedContentSchema(schema)
	return schema
}

func setPagedContentSchema(schema *jsonschema.Schema) {
	schema.Properties["page"].Minimum = jsonschema.Ptr(float64(1))
	schema.Properties["page"].Default = json.RawMessage(`1`)
	schema.Properties["page_size"].Minimum = jsonschema.Ptr(float64(1))
	schema.Properties["page_size"].Maximum = jsonschema.Ptr(float64(20))
	schema.AllOf = append(schema.AllOf, &jsonschema.Schema{
		If: &jsonschema.Schema{
			Required: []string{"content_type"},
			Properties: map[string]*jsonschema.Schema{
				"content_type": {Enum: []any{"article", "skill", "mcp_server"}},
			},
		},
		Then: &jsonschema.Schema{Properties: map[string]*jsonschema.Schema{
			"page_size": {Maximum: jsonschema.Ptr(float64(20)), Default: json.RawMessage(`10`)},
		}},
		Else: &jsonschema.Schema{Properties: map[string]*jsonschema.Schema{
			"page_size": {Maximum: jsonschema.Ptr(float64(10)), Default: json.RawMessage(`5`)},
		}},
	})
}

func searchContent(deps PublicDependencies, publicBaseURL string) mcp.ToolHandlerFor[searchContentInput, searchContentOutput] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input searchContentInput) (*mcp.CallToolResult, searchContentOutput, error) {
		contentType, sort, page, pageSize, err := normalizeSearchInput(&input)
		if err != nil {
			return nil, searchContentOutput{}, err
		}
		output := searchContentOutput{
			ContentType: contentType,
			Sort:        sort,
			Articles:    []contentSummaryOutput{},
			Skills:      []contentSummaryOutput{},
			MCPServers:  []contentSummaryOutput{},
			PageInfo:    PageInfo{Page: page, PageSize: pageSize},
		}

		if sort == "relevance" {
			if deps.Search == nil {
				return nil, searchContentOutput{}, internalError()
			}
			response, err := deps.Search.Search(ctx, service.SearchQuery{
				Keyword: strings.TrimSpace(input.Query), ContentType: contentType,
				TagID: input.TagID, CategoryID: input.CategoryID,
				Page: page, PageSize: pageSize, Highlight: false,
			})
			if err != nil || response == nil {
				return nil, searchContentOutput{}, internalError()
			}
			output.Articles = searchResultsOutput(response.Articles, publicBaseURL)
			output.Skills = searchResultsOutput(response.Skills, publicBaseURL)
			output.MCPServers = searchResultsOutput(response.McpServers, publicBaseURL)
			output.Total = response.Total
		} else if err := searchLatest(ctx, deps, input, publicBaseURL, &output); err != nil {
			return nil, searchContentOutput{}, err
		}

		count := len(output.Articles) + len(output.Skills) + len(output.MCPServers)
		return summaryResult(fmt.Sprintf("Search returned %d published result(s).", count)), output, nil
	}
}

func normalizeSearchInput(input *searchContentInput) (contentType, sort string, page, pageSize int, err error) {
	contentType = strings.TrimSpace(input.ContentType)
	if contentType == "" {
		contentType = "all"
	}
	if !validContentType(contentType) {
		return "", "", 0, 0, invalidArgument("content_type must be all, article, skill, or mcp_server")
	}
	input.Query = strings.TrimSpace(input.Query)
	if input.TagID != nil && *input.TagID == 0 {
		return "", "", 0, 0, invalidArgument("tag_id must be greater than zero")
	}
	if input.CategoryID != nil && *input.CategoryID == 0 {
		return "", "", 0, 0, invalidArgument("category_id must be greater than zero")
	}
	if input.Query == "" && input.TagID == nil && input.CategoryID == nil {
		return "", "", 0, 0, invalidArgument("query, tag_id, or category_id is required")
	}
	if input.CategoryID != nil && contentType != "article" {
		return "", "", 0, 0, invalidArgument("category_id is only valid for article content")
	}

	sort = strings.TrimSpace(input.Sort)
	if sort == "" {
		if input.Query == "" {
			sort = "latest"
		} else {
			sort = "relevance"
		}
	}
	if sort != "relevance" && sort != "latest" {
		return "", "", 0, 0, invalidArgument("sort must be relevance or latest")
	}
	if sort == "relevance" && input.Query == "" {
		return "", "", 0, 0, invalidArgument("relevance sort requires query")
	}

	page, pageSize, err = normalizePage(input.Page, input.PageSize, contentType)
	if err != nil {
		return "", "", 0, 0, err
	}
	return contentType, sort, page, pageSize, nil
}

func searchLatest(ctx context.Context, deps PublicDependencies, input searchContentInput, publicBaseURL string, output *searchContentOutput) error {
	query := strings.TrimSpace(input.Query)
	switch output.ContentType {
	case "article":
		if deps.Articles == nil {
			return internalError()
		}
		result, err := deps.Articles.List(ctx, service.ListQuery{
			Page: output.Page, PageSize: output.PageSize, Keyword: query,
			TagID: input.TagID, CategoryID: input.CategoryID, Sort: "latest",
		})
		if err != nil || result == nil {
			return internalError()
		}
		output.Articles = articleSummariesOutput(result.List, publicBaseURL)
		output.Total = result.Total
	case "skill":
		if deps.Skills == nil {
			return internalError()
		}
		result, err := deps.Skills.List(ctx, service.SkillListQuery{
			Page: output.Page, PageSize: output.PageSize, Keyword: query, TagID: input.TagID, Sort: "latest",
		})
		if err != nil || result == nil {
			return internalError()
		}
		output.Skills = skillSummariesOutput(result.List, publicBaseURL)
		output.Total = result.Total
	case "mcp_server":
		if deps.MCPServers == nil {
			return internalError()
		}
		result, err := deps.MCPServers.List(ctx, service.McpServerListQuery{
			Page: output.Page, PageSize: output.PageSize, Keyword: query, TagID: input.TagID, Sort: "latest",
		})
		if err != nil || result == nil {
			return internalError()
		}
		output.MCPServers = mcpServerSummariesOutput(result.List, publicBaseURL)
		output.Total = result.Total
	case "all":
		if deps.Articles == nil || deps.Skills == nil || deps.MCPServers == nil {
			return internalError()
		}
		articles, err := deps.Articles.List(ctx, service.ListQuery{
			Page: output.Page, PageSize: output.PageSize, Keyword: query, TagID: input.TagID, Sort: "latest",
		})
		if err != nil || articles == nil {
			return internalError()
		}
		skills, err := deps.Skills.List(ctx, service.SkillListQuery{
			Page: output.Page, PageSize: output.PageSize, Keyword: query, TagID: input.TagID, Sort: "latest",
		})
		if err != nil || skills == nil {
			return internalError()
		}
		servers, err := deps.MCPServers.List(ctx, service.McpServerListQuery{
			Page: output.Page, PageSize: output.PageSize, Keyword: query, TagID: input.TagID, Sort: "latest",
		})
		if err != nil || servers == nil {
			return internalError()
		}
		output.Articles = articleSummariesOutput(articles.List, publicBaseURL)
		output.Skills = skillSummariesOutput(skills.List, publicBaseURL)
		output.MCPServers = mcpServerSummariesOutput(servers.List, publicBaseURL)
		output.Total = articles.Total + skills.Total + servers.Total
	}
	return nil
}

func browseContent(deps PublicDependencies, publicBaseURL string) mcp.ToolHandlerFor[browseContentInput, browseContentOutput] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input browseContentInput) (*mcp.CallToolResult, browseContentOutput, error) {
		contentType := strings.TrimSpace(input.ContentType)
		if contentType == "" {
			contentType = "all"
		}
		if !validContentType(contentType) {
			return nil, browseContentOutput{}, invalidArgument("content_type must be all, article, skill, or mcp_server")
		}
		sort := strings.TrimSpace(input.Sort)
		if sort == "" {
			sort = "latest"
		}
		if sort != "latest" && sort != "hot" && sort != "downloads" {
			return nil, browseContentOutput{}, invalidArgument("sort must be latest, hot, or downloads")
		}
		if sort == "downloads" && contentType != "skill" && contentType != "mcp_server" {
			return nil, browseContentOutput{}, invalidArgument("downloads sort is only valid for skill or mcp_server")
		}
		page, pageSize, err := normalizePage(input.Page, input.PageSize, contentType)
		if err != nil {
			return nil, browseContentOutput{}, err
		}
		output := browseContentOutput{
			ContentType: contentType, Sort: sort,
			Articles: []contentSummaryOutput{}, Skills: []contentSummaryOutput{}, MCPServers: []contentSummaryOutput{},
			PageInfo: PageInfo{Page: page, PageSize: pageSize},
		}
		if sort == "hot" {
			if err := browseHot(ctx, deps, publicBaseURL, &output); err != nil {
				return nil, browseContentOutput{}, err
			}
		} else if err := browseListed(ctx, deps, publicBaseURL, &output); err != nil {
			return nil, browseContentOutput{}, err
		}
		count := len(output.Articles) + len(output.Skills) + len(output.MCPServers)
		return summaryResult(fmt.Sprintf("Browse returned %d published result(s).", count)), output, nil
	}
}

func browseListed(ctx context.Context, deps PublicDependencies, publicBaseURL string, output *browseContentOutput) error {
	switch output.ContentType {
	case "article":
		if deps.Articles == nil {
			return internalError()
		}
		result, err := deps.Articles.List(ctx, service.ListQuery{Page: output.Page, PageSize: output.PageSize, Sort: output.Sort})
		if err != nil || result == nil {
			return internalError()
		}
		output.Articles = articleSummariesOutput(result.List, publicBaseURL)
		output.Total = result.Total
	case "skill":
		if deps.Skills == nil {
			return internalError()
		}
		result, err := deps.Skills.List(ctx, service.SkillListQuery{Page: output.Page, PageSize: output.PageSize, Sort: output.Sort})
		if err != nil || result == nil {
			return internalError()
		}
		output.Skills = skillSummariesOutput(result.List, publicBaseURL)
		output.Total = result.Total
	case "mcp_server":
		if deps.MCPServers == nil {
			return internalError()
		}
		result, err := deps.MCPServers.List(ctx, service.McpServerListQuery{Page: output.Page, PageSize: output.PageSize, Sort: output.Sort})
		if err != nil || result == nil {
			return internalError()
		}
		output.MCPServers = mcpServerSummariesOutput(result.List, publicBaseURL)
		output.Total = result.Total
	case "all":
		if deps.Articles == nil || deps.Skills == nil || deps.MCPServers == nil {
			return internalError()
		}
		articles, err := deps.Articles.List(ctx, service.ListQuery{Page: output.Page, PageSize: output.PageSize, Sort: output.Sort})
		if err != nil || articles == nil {
			return internalError()
		}
		skills, err := deps.Skills.List(ctx, service.SkillListQuery{Page: output.Page, PageSize: output.PageSize, Sort: output.Sort})
		if err != nil || skills == nil {
			return internalError()
		}
		servers, err := deps.MCPServers.List(ctx, service.McpServerListQuery{Page: output.Page, PageSize: output.PageSize, Sort: output.Sort})
		if err != nil || servers == nil {
			return internalError()
		}
		output.Articles = articleSummariesOutput(articles.List, publicBaseURL)
		output.Skills = skillSummariesOutput(skills.List, publicBaseURL)
		output.MCPServers = mcpServerSummariesOutput(servers.List, publicBaseURL)
		output.Total = articles.Total + skills.Total + servers.Total
	}
	return nil
}

func browseHot(ctx context.Context, deps PublicDependencies, publicBaseURL string, output *browseContentOutput) error {
	if deps.Ranking == nil {
		return temporarilyUnavailable()
	}
	if output.ContentType == "article" || output.ContentType == "all" {
		articles, err := deps.Ranking.ListArticleHot(ctx, output.Page, output.PageSize)
		if err != nil {
			return temporarilyUnavailable()
		}
		output.Articles = articleSummariesOutput(articles, publicBaseURL)
	}
	if output.ContentType == "skill" || output.ContentType == "all" {
		skills, err := deps.Ranking.ListSkillHot(ctx, output.Page, output.PageSize)
		if err != nil {
			return temporarilyUnavailable()
		}
		output.Skills = skillSummariesOutput(skills, publicBaseURL)
	}
	if output.ContentType == "mcp_server" || output.ContentType == "all" {
		servers, err := deps.Ranking.ListMcpServerHot(ctx, output.Page, output.PageSize)
		if err != nil {
			return temporarilyUnavailable()
		}
		output.MCPServers = mcpServerSummariesOutput(servers, publicBaseURL)
	}
	output.Total = int64(len(output.Articles) + len(output.Skills) + len(output.MCPServers))
	return nil
}

func validContentType(contentType string) bool {
	switch contentType {
	case "all", "article", "skill", "mcp_server":
		return true
	default:
		return false
	}
}

func normalizePage(page, pageSize int, contentType string) (int, int, error) {
	if page < 0 || pageSize < 0 {
		return 0, 0, invalidArgument("page and page_size must be positive")
	}
	if page == 0 {
		page = 1
	}
	defaultPageSize, maxPageSize := 10, 20
	if contentType == "all" {
		defaultPageSize, maxPageSize = 5, 10
	}
	if pageSize == 0 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		return 0, 0, invalidArgument(fmt.Sprintf("page_size must not exceed %d for %s", maxPageSize, contentType))
	}
	return page, pageSize, nil
}

func searchResultsOutput(results []service.SearchResult, publicBaseURL string) []contentSummaryOutput {
	output := make([]contentSummaryOutput, 0, len(results))
	for _, result := range results {
		output = append(output, contentSummaryOutput{
			ID: result.ID, Type: result.Type, Title: result.Title, Summary: result.Summary,
			URL: contentPageURL(publicBaseURL, result.Type, result.ID), Author: authorOutput(result.Author, publicBaseURL),
			Tags: tagOutputs(result.Tags), Views: result.Views,
		})
	}
	return output
}

func articleSummariesOutput(summaries []service.ArticleSummary, publicBaseURL string) []contentSummaryOutput {
	output := make([]contentSummaryOutput, 0, len(summaries))
	for _, summary := range summaries {
		output = append(output, contentSummaryOutput{
			ID: summary.ID, Type: "article", Title: summary.Title, Summary: summary.Summary,
			URL:    contentPageURL(publicBaseURL, "article", summary.ID),
			Author: authorOutput(summary.Author, publicBaseURL), Tags: tagOutputs(summary.Tags),
			Views: summary.Views, PublishedAt: publishedAtOutput(summary.PublishedAt),
		})
	}
	return output
}

func skillSummariesOutput(summaries []service.SkillSummary, publicBaseURL string) []contentSummaryOutput {
	output := make([]contentSummaryOutput, 0, len(summaries))
	for _, summary := range summaries {
		output = append(output, contentSummaryOutput{
			ID: summary.ID, Type: "skill", Title: summary.Name, Summary: summary.Description,
			URL:    contentPageURL(publicBaseURL, "skill", summary.ID),
			Author: authorOutput(summary.Author, publicBaseURL), Tags: tagOutputs(summary.Tags),
			Views: summary.Views, Downloads: summary.Downloads, PublishedAt: publishedAtOutput(summary.PublishedAt),
		})
	}
	return output
}

func mcpServerSummariesOutput(summaries []service.McpServerSummary, publicBaseURL string) []contentSummaryOutput {
	output := make([]contentSummaryOutput, 0, len(summaries))
	for _, summary := range summaries {
		output = append(output, contentSummaryOutput{
			ID: summary.ID, Type: "mcp_server", Title: summary.Name, Summary: summary.Description,
			URL:    contentPageURL(publicBaseURL, "mcp_server", summary.ID),
			Author: authorOutput(summary.Author, publicBaseURL), Tags: tagOutputs(summary.Tags),
			Views: summary.Views, Downloads: summary.Downloads, PublishedAt: publishedAtOutput(summary.PublishedAt),
		})
	}
	return output
}

func summaryResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}
}
