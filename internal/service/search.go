package service

import (
	"context"
	"html"
	"regexp"
	"strings"

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

type SearchResponse struct {
	Items    []SearchResult   `json:"items"`
	Total    int64            `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"page_size"`
	Counts   map[string]int64 `json:"counts,omitempty"`
}

func (s *SearchService) Search(ctx context.Context, keyword, searchType string, tagID, categoryID *uint, page, pageSize int) (*SearchResponse, error) {
	if pageSize == 0 {
		pageSize = 20
	}

	var items []SearchResult
	var total int64
	counts := make(map[string]int64)

	switch searchType {
	case "article":
		articles, count, err := s.searchRepo.SearchArticles(ctx, keyword, tagID, categoryID, page, pageSize)
		if err != nil {
			return nil, err
		}
		total = count
		for _, a := range articles {
			items = append(items, SearchResult{
				ID:      a.ID,
				Type:    "article",
				Title:   highlightText(a.Title, keyword),
				Summary: highlightText(a.Summary, keyword),
				Views:   a.Views,
			})
		}

	case "skill":
		skills, count, err := s.searchRepo.SearchSkills(ctx, keyword, tagID, page, pageSize)
		if err != nil {
			return nil, err
		}
		total = count
		for _, sk := range skills {
			items = append(items, SearchResult{
				ID:      sk.ID,
				Type:    "skill",
				Title:   highlightText(sk.Name, keyword),
				Summary: highlightText(sk.Description, keyword),
				Views:   sk.Views,
			})
		}

	case "mcp_server":
		servers, count, err := s.searchRepo.SearchMcpServers(ctx, keyword, tagID, page, pageSize)
		if err != nil {
			return nil, err
		}
		total = count
		for _, sv := range servers {
			items = append(items, SearchResult{
				ID:      sv.ID,
				Type:    "mcp_server",
				Title:   highlightText(sv.Name, keyword),
				Summary: highlightText(sv.Description, keyword),
				Views:   sv.Views,
			})
		}

	default:
		_, articleCount, _ := s.searchRepo.SearchArticles(ctx, keyword, tagID, categoryID, 1, 1)
		_, skillCount, _ := s.searchRepo.SearchSkills(ctx, keyword, tagID, 1, 1)
		_, mcpCount, _ := s.searchRepo.SearchMcpServers(ctx, keyword, tagID, 1, 1)

		counts["article"] = articleCount
		counts["skill"] = skillCount
		counts["mcp_server"] = mcpCount
		total = articleCount + skillCount + mcpCount

		articles, _, _ := s.searchRepo.SearchArticles(ctx, keyword, tagID, categoryID, page, pageSize)
		for _, a := range articles {
			items = append(items, SearchResult{
				ID:      a.ID,
				Type:    "article",
				Title:   highlightText(a.Title, keyword),
				Summary: highlightText(a.Summary, keyword),
				Views:   a.Views,
			})
		}
	}

	return &SearchResponse{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		Counts:   counts,
	}, nil
}

func HighlightText(text, keyword string) string {
	return highlightText(text, keyword)
}

func init() {
	_ = strings.NewReader
}
