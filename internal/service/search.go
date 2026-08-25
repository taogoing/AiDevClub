package service

import (
	"context"
	"html"
	"regexp"

	"aidevclub/internal/model"
	"aidevclub/internal/repo"
)

type SearchService struct {
	searchRepo *repo.SearchRepo
}

func NewSearchService(searchRepo *repo.SearchRepo) *SearchService {
	return &SearchService{searchRepo: searchRepo}
}

func highlightText(text, keyword string) string {
	if keyword == "" || text == "" {
		return text
	}

	escaped := html.EscapeString(text)
	pattern := regexp.MustCompile(`(?i)` + regexp.QuoteMeta(keyword))
	return pattern.ReplaceAllStringFunc(escaped, func(match string) string {
		return "<mark>" + match + "</mark>"
	})
}

type SearchResult struct {
	ID         uint        `json:"id"`
	Type       string      `json:"type"`
	Title      string      `json:"title"`
	Summary    string      `json:"summary"`
	Author     interface{} `json:"author"`
	Tags       []model.Tag `json:"tags"`
	Views      int         `json:"views"`
	LikesCount int         `json:"likes_count"`
	CreatedAt  string      `json:"created_at"`
}

type SearchQuery struct {
	Keyword     string
	ContentType string
	// Type is retained as a compatibility alias for callers that adopted the
	// original reader sketch before ContentType was finalized.
	Type       string
	Sort       string
	TagID      *uint
	CategoryID *uint
	Page       int
	PageSize   int
	Highlight  bool
}

type SearchResponse struct {
	Items      []SearchResult   `json:"items"`
	Articles   []SearchResult   `json:"articles"`
	Skills     []SearchResult   `json:"skills"`
	McpServers []SearchResult   `json:"mcp_servers"`
	Total      int64            `json:"total"`
	Page       int              `json:"page"`
	PageSize   int              `json:"page_size"`
	Counts     map[string]int64 `json:"counts,omitempty"`
}

func (s *SearchService) Search(ctx context.Context, q SearchQuery) (*SearchResponse, error) {
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 {
		q.PageSize = 20
	}
	contentType := q.ContentType
	if contentType == "" {
		contentType = q.Type
	}

	response := &SearchResponse{
		Items:      []SearchResult{},
		Articles:   []SearchResult{},
		Skills:     []SearchResult{},
		McpServers: []SearchResult{},
		Page:       q.Page,
		PageSize:   q.PageSize,
		Counts:     make(map[string]int64),
	}

	switch contentType {
	case "article":
		articles, count, err := s.searchRepo.SearchArticles(ctx, q.Keyword, q.TagID, q.CategoryID, q.Page, q.PageSize)
		if err != nil {
			return nil, err
		}
		response.Total = count
		response.Counts["article"] = count
		for _, a := range articles {
			item := SearchResult{
				ID:      a.ID,
				Type:    "article",
				Title:   searchText(a.Title, q.Keyword, q.Highlight),
				Summary: searchText(a.Summary, q.Keyword, q.Highlight),
				Views:   a.Views,
			}
			response.Articles = append(response.Articles, item)
			response.Items = append(response.Items, item)
		}

	case "skill":
		skills, count, err := s.searchRepo.SearchSkills(ctx, q.Keyword, q.TagID, q.Page, q.PageSize)
		if err != nil {
			return nil, err
		}
		response.Total = count
		response.Counts["skill"] = count
		for _, sk := range skills {
			item := SearchResult{
				ID:      sk.ID,
				Type:    "skill",
				Title:   searchText(sk.Name, q.Keyword, q.Highlight),
				Summary: searchText(sk.Description, q.Keyword, q.Highlight),
				Views:   sk.Views,
			}
			response.Skills = append(response.Skills, item)
			response.Items = append(response.Items, item)
		}

	case "mcp_server":
		servers, count, err := s.searchRepo.SearchMcpServers(ctx, q.Keyword, q.TagID, q.Page, q.PageSize)
		if err != nil {
			return nil, err
		}
		response.Total = count
		response.Counts["mcp_server"] = count
		for _, sv := range servers {
			item := SearchResult{
				ID:      sv.ID,
				Type:    "mcp_server",
				Title:   searchText(sv.Name, q.Keyword, q.Highlight),
				Summary: searchText(sv.Description, q.Keyword, q.Highlight),
				Views:   sv.Views,
			}
			response.McpServers = append(response.McpServers, item)
			response.Items = append(response.Items, item)
		}

	default:
		articles, articleCount, err := s.searchRepo.SearchArticles(ctx, q.Keyword, q.TagID, q.CategoryID, q.Page, q.PageSize)
		if err != nil {
			return nil, err
		}
		skills, skillCount, err := s.searchRepo.SearchSkills(ctx, q.Keyword, q.TagID, q.Page, q.PageSize)
		if err != nil {
			return nil, err
		}
		servers, mcpCount, err := s.searchRepo.SearchMcpServers(ctx, q.Keyword, q.TagID, q.Page, q.PageSize)
		if err != nil {
			return nil, err
		}

		response.Counts["article"] = articleCount
		response.Counts["skill"] = skillCount
		response.Counts["mcp_server"] = mcpCount
		response.Total = articleCount + skillCount + mcpCount
		for _, a := range articles {
			response.Articles = append(response.Articles, SearchResult{
				ID:      a.ID,
				Type:    "article",
				Title:   searchText(a.Title, q.Keyword, q.Highlight),
				Summary: searchText(a.Summary, q.Keyword, q.Highlight),
				Views:   a.Views,
			})
		}
		for _, sk := range skills {
			response.Skills = append(response.Skills, SearchResult{
				ID: sk.ID, Type: "skill", Title: searchText(sk.Name, q.Keyword, q.Highlight),
				Summary: searchText(sk.Description, q.Keyword, q.Highlight), Views: sk.Views,
			})
		}
		for _, sv := range servers {
			response.McpServers = append(response.McpServers, SearchResult{
				ID: sv.ID, Type: "mcp_server", Title: searchText(sv.Name, q.Keyword, q.Highlight),
				Summary: searchText(sv.Description, q.Keyword, q.Highlight), Views: sv.Views,
			})
		}
		// The typed sections are the MCP-facing all-type contract. Items remains
		// a deterministic, type-grouped projection for the existing web search
		// page; it does not claim a cross-type relevance order.
		response.Items = append(response.Items, response.Articles...)
		response.Items = append(response.Items, response.Skills...)
		response.Items = append(response.Items, response.McpServers...)
	}

	return response, nil
}

func searchText(text, keyword string, highlight bool) string {
	if !highlight {
		return text
	}
	return highlightText(text, keyword)
}

func HighlightText(text, keyword string) string {
	return highlightText(text, keyword)
}
