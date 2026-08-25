package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"aidevclub/internal/service"
)

const (
	defaultContentLimit = 30000
	maximumContentLimit = 50000
)

type ContentWindowInput struct {
	ContentOffset int `json:"content_offset,omitempty" jsonschema:"Unicode character offset, minimum 0."`
	ContentLimit  int `json:"content_limit,omitempty" jsonschema:"Unicode character limit, from 1 through 50000."`
}

type getArticleInput struct {
	ID uint `json:"id" jsonschema:"Published article ID."`
	ContentWindowInput
}

type getArticleOutput struct {
	ID             uint         `json:"id"`
	Title          string       `json:"title"`
	Summary        string       `json:"summary"`
	CategoryID     uint         `json:"category_id"`
	CategoryName   string       `json:"category_name"`
	Tags           []TagOutput  `json:"tags"`
	Author         AuthorOutput `json:"author"`
	Views          int          `json:"views"`
	LikesCount     int          `json:"likes_count"`
	FavoritesCount int          `json:"favorites_count"`
	CommentsCount  int          `json:"comments_count"`
	PublishedAt    string       `json:"published_at,omitempty"`
	URL            string       `json:"url"`
	Content        string       `json:"content"`
	HasMore        bool         `json:"has_more"`
	NextOffset     int          `json:"next_offset"`
}

type getSkillInput struct {
	ID uint `json:"id" jsonschema:"Published Skill ID."`
	ContentWindowInput
}

type getSkillOutput struct {
	ID                uint         `json:"id"`
	Name              string       `json:"name"`
	Description       string       `json:"description"`
	RepoURL           string       `json:"repo_url,omitempty"`
	Tags              []TagOutput  `json:"tags"`
	Author            AuthorOutput `json:"author"`
	Views             int          `json:"views"`
	Downloads         int          `json:"downloads"`
	LikesCount        int          `json:"likes_count"`
	FavoritesCount    int          `json:"favorites_count"`
	CommentsCount     int          `json:"comments_count"`
	PublishedAt       string       `json:"published_at,omitempty"`
	URL               string       `json:"url"`
	DownloadAvailable bool         `json:"download_available"`
	Filename          string       `json:"filename,omitempty"`
	FileSize          int64        `json:"file_size"`
	SkillMD           string       `json:"skill_md"`
	HasMore           bool         `json:"has_more"`
	NextOffset        int          `json:"next_offset"`
}

type getMCPServerInput struct {
	ID uint `json:"id" jsonschema:"Published MCP Server ID."`
	ContentWindowInput
}

type getMCPServerOutput struct {
	ID                uint         `json:"id"`
	Name              string       `json:"name"`
	Description       string       `json:"description"`
	RepoURL           string       `json:"repo_url,omitempty"`
	Tags              []TagOutput  `json:"tags"`
	Author            AuthorOutput `json:"author"`
	Views             int          `json:"views"`
	Downloads         int          `json:"downloads"`
	LikesCount        int          `json:"likes_count"`
	FavoritesCount    int          `json:"favorites_count"`
	CommentsCount     int          `json:"comments_count"`
	PublishedAt       string       `json:"published_at,omitempty"`
	URL               string       `json:"url"`
	DownloadAvailable bool         `json:"download_available"`
	Filename          string       `json:"filename,omitempty"`
	FileSize          int64        `json:"file_size"`
	ToolsJSON         any          `json:"tools_json"`
	Readme            string       `json:"readme"`
	HasMore           bool         `json:"has_more"`
	NextOffset        int          `json:"next_offset"`
}

func getArticleInputSchema() *jsonschema.Schema {
	schema := mustInputSchema[getArticleInput]()
	setContentWindowInputSchema(schema)
	return schema
}

func getSkillInputSchema() *jsonschema.Schema {
	schema := mustInputSchema[getSkillInput]()
	setContentWindowInputSchema(schema)
	return schema
}

func getMCPServerInputSchema() *jsonschema.Schema {
	schema := mustInputSchema[getMCPServerInput]()
	setContentWindowInputSchema(schema)
	return schema
}

func setContentWindowInputSchema(schema *jsonschema.Schema) {
	schema.Properties["id"].Minimum = jsonschema.Ptr(float64(1))
	schema.Properties["content_offset"].Minimum = jsonschema.Ptr(float64(0))
	schema.Properties["content_offset"].Default = json.RawMessage(`0`)
	schema.Properties["content_limit"].Minimum = jsonschema.Ptr(float64(1))
	schema.Properties["content_limit"].Maximum = jsonschema.Ptr(float64(maximumContentLimit))
	schema.Properties["content_limit"].Default = json.RawMessage(`30000`)
}

func getArticle(reader ArticleReader, publicBaseURL string) mcp.ToolHandlerFor[getArticleInput, getArticleOutput] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input getArticleInput) (*mcp.CallToolResult, getArticleOutput, error) {
		offset, limit, err := normalizeContentWindow(input.ID, input.ContentWindowInput)
		if err != nil {
			return nil, getArticleOutput{}, err
		}
		if reader == nil {
			return nil, getArticleOutput{}, internalError()
		}
		detail, err := reader.Read(ctx, 0, input.ID)
		if err != nil {
			return nil, getArticleOutput{}, contentReadError(err, service.ErrArticleNotFound)
		}
		if detail == nil {
			return nil, getArticleOutput{}, internalError()
		}
		window := unicodeWindow(detail.Content, offset, limit)
		output := getArticleOutput{
			ID: detail.ID, Title: detail.Title, Summary: detail.Summary,
			CategoryID: detail.CategoryID, CategoryName: detail.CategoryName,
			Tags: tagOutputs(detail.Tags), Author: authorOutput(detail.Author, publicBaseURL),
			Views: detail.Views, LikesCount: detail.LikesCount, FavoritesCount: detail.FavoritesCount,
			CommentsCount: detail.CommentsCount, PublishedAt: publishedAtOutput(detail.PublishedAt),
			URL:     contentPageURL(publicBaseURL, "article", detail.ID),
			Content: window.Text, HasMore: window.HasMore, NextOffset: window.NextOffset,
		}
		return summaryResult(fmt.Sprintf("Article %d returned %d Unicode character(s).", detail.ID, len([]rune(window.Text)))), output, nil
	}
}

func getSkill(reader SkillReader, publicBaseURL string) mcp.ToolHandlerFor[getSkillInput, getSkillOutput] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input getSkillInput) (*mcp.CallToolResult, getSkillOutput, error) {
		offset, limit, err := normalizeContentWindow(input.ID, input.ContentWindowInput)
		if err != nil {
			return nil, getSkillOutput{}, err
		}
		if reader == nil {
			return nil, getSkillOutput{}, internalError()
		}
		detail, err := reader.Read(ctx, 0, input.ID)
		if err != nil {
			return nil, getSkillOutput{}, contentReadError(err, service.ErrSkillNotFound)
		}
		if detail == nil {
			return nil, getSkillOutput{}, internalError()
		}
		window := unicodeWindow(detail.SkillMD, offset, limit)
		filename := archiveFilenameOutput(detail.ZipFilename)
		output := getSkillOutput{
			ID: detail.ID, Name: detail.Name, Description: detail.Description,
			RepoURL: absoluteExternalURL(detail.RepoURL),
			Tags:    tagOutputs(detail.Tags), Author: authorOutput(detail.Author, publicBaseURL),
			Views: detail.Views, Downloads: detail.Downloads, LikesCount: detail.LikesCount,
			FavoritesCount: detail.FavoritesCount, CommentsCount: detail.CommentsCount,
			PublishedAt: publishedAtOutput(detail.PublishedAt), URL: contentPageURL(publicBaseURL, "skill", detail.ID),
			DownloadAvailable: detail.ZipURL != "" && filename != "" && detail.FileSize > 0,
			Filename:          filename, FileSize: detail.FileSize,
			SkillMD: window.Text, HasMore: window.HasMore, NextOffset: window.NextOffset,
		}
		return summaryResult(fmt.Sprintf("Skill %d returned %d Unicode character(s) from SKILL.md.", detail.ID, len([]rune(window.Text)))), output, nil
	}
}

func getMCPServer(reader MCPServerReader, publicBaseURL string) mcp.ToolHandlerFor[getMCPServerInput, getMCPServerOutput] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input getMCPServerInput) (*mcp.CallToolResult, getMCPServerOutput, error) {
		offset, limit, err := normalizeContentWindow(input.ID, input.ContentWindowInput)
		if err != nil {
			return nil, getMCPServerOutput{}, err
		}
		if reader == nil {
			return nil, getMCPServerOutput{}, internalError()
		}
		detail, err := reader.Read(ctx, 0, input.ID)
		if err != nil {
			return nil, getMCPServerOutput{}, contentReadError(err, service.ErrMcpServerNotFound)
		}
		if detail == nil {
			return nil, getMCPServerOutput{}, internalError()
		}
		toolsSource := strings.TrimSpace(detail.ToolsJSON)
		if toolsSource == "" {
			toolsSource = "[]"
		}
		var tools any
		if err := json.Unmarshal([]byte(toolsSource), &tools); err != nil {
			return nil, getMCPServerOutput{}, internalError()
		}
		window := unicodeWindow(detail.Readme, offset, limit)
		filename := archiveFilenameOutput(detail.ZipFilename)
		output := getMCPServerOutput{
			ID: detail.ID, Name: detail.Name, Description: detail.Description,
			RepoURL: absoluteExternalURL(detail.RepoURL),
			Tags:    tagOutputs(detail.Tags), Author: authorOutput(detail.Author, publicBaseURL),
			Views: detail.Views, Downloads: detail.Downloads, LikesCount: detail.LikesCount,
			FavoritesCount: detail.FavoritesCount, CommentsCount: detail.CommentsCount,
			PublishedAt: publishedAtOutput(detail.PublishedAt), URL: contentPageURL(publicBaseURL, "mcp_server", detail.ID),
			DownloadAvailable: detail.ZipURL != "" && filename != "" && detail.FileSize > 0,
			Filename:          filename, FileSize: detail.FileSize,
			ToolsJSON: tools, Readme: window.Text, HasMore: window.HasMore, NextOffset: window.NextOffset,
		}
		return summaryResult(fmt.Sprintf("MCP Server %d returned %d Unicode character(s) from README.", detail.ID, len([]rune(window.Text)))), output, nil
	}
}

func normalizeContentWindow(id uint, input ContentWindowInput) (offset, limit int, err error) {
	if id == 0 {
		return 0, 0, invalidArgument("id must be greater than zero")
	}
	if input.ContentOffset < 0 {
		return 0, 0, invalidArgument("content_offset must not be negative")
	}
	if input.ContentLimit < 0 {
		return 0, 0, invalidArgument("content_limit must be positive")
	}
	if input.ContentLimit > maximumContentLimit {
		return 0, 0, invalidArgument("content_limit must not exceed 50000")
	}
	limit = input.ContentLimit
	if limit == 0 {
		limit = defaultContentLimit
	}
	return input.ContentOffset, limit, nil
}

func absoluteExternalURL(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		return value
	}
	return ""
}

func archiveFilenameOutput(value string) string {
	normalized := strings.ReplaceAll(strings.TrimSpace(value), `\`, "/")
	if normalized == "" || strings.HasSuffix(normalized, "/") {
		return ""
	}
	filename := path.Base(normalized)
	if filename == "." || filename == ".." || filename == "/" {
		return ""
	}
	return filename
}
