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
	Author     AuthorBrief `json:"author"`
	Tags       []TagBrief  `json:"tags"`
	Views      int         `json:"views"`
	LikesCount int         `json:"likes_count"`
	CreatedAt  string      `json:"created_at"`
}

type SearchQuery struct {
	Keyword     string
	ContentType string
	TagID       *uint
	Page        int
	PageSize    int
	Highlight   bool
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
	response := &SearchResponse{
		Items:      []SearchResult{},
		Articles:   []SearchResult{},
		Skills:     []SearchResult{},
		McpServers: []SearchResult{},
		Page:       q.Page,
		PageSize:   q.PageSize,
	}

	switch q.ContentType {
	case "article":
		articles, count, err := s.searchRepo.SearchArticles(ctx, q.Keyword, q.TagID, q.Page, q.PageSize)
		if err != nil {
			return nil, err
		}
		response.Total = count
		items, err := s.articleSearchResults(ctx, articles, q)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			response.Articles = append(response.Articles, item)
			response.Items = append(response.Items, item)
		}

	case "skill":
		skills, count, err := s.searchRepo.SearchSkills(ctx, q.Keyword, q.TagID, q.Page, q.PageSize)
		if err != nil {
			return nil, err
		}
		response.Total = count
		items, err := s.skillSearchResults(ctx, skills, q)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			response.Skills = append(response.Skills, item)
			response.Items = append(response.Items, item)
		}

	case "mcp_server":
		servers, count, err := s.searchRepo.SearchMcpServers(ctx, q.Keyword, q.TagID, q.Page, q.PageSize)
		if err != nil {
			return nil, err
		}
		response.Total = count
		items, err := s.mcpServerSearchResults(ctx, servers, q)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			response.McpServers = append(response.McpServers, item)
			response.Items = append(response.Items, item)
		}

	default:
		articles, articleCount, err := s.searchRepo.SearchArticles(ctx, q.Keyword, q.TagID, q.Page, q.PageSize)
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

		response.Counts = map[string]int64{
			"article":    articleCount,
			"skill":      skillCount,
			"mcp_server": mcpCount,
		}
		response.Total = articleCount + skillCount + mcpCount
		response.Articles, err = s.articleSearchResults(ctx, articles, q)
		if err != nil {
			return nil, err
		}
		response.Skills, err = s.skillSearchResults(ctx, skills, q)
		if err != nil {
			return nil, err
		}
		response.McpServers, err = s.mcpServerSearchResults(ctx, servers, q)
		if err != nil {
			return nil, err
		}
		// Typed sections expose grouped MCP search results. Items preserves the
		// legacy REST contract: the requested standard page of articles only.
		response.Items = append(response.Items, response.Articles...)
	}

	return response, nil
}

func (s *SearchService) articleSearchResults(ctx context.Context, articles []model.Article, q SearchQuery) ([]SearchResult, error) {
	ids := make([]uint, 0, len(articles))
	for _, article := range articles {
		ids = append(ids, article.ID)
	}
	tagMap, err := s.searchRepo.TagsForArticles(ctx, ids)
	if err != nil {
		return nil, err
	}
	results := make([]SearchResult, 0, len(articles))
	for _, article := range articles {
		results = append(results, SearchResult{
			ID: article.ID, Type: "article",
			Title: searchText(article.Title, q.Keyword, q.Highlight), Summary: searchText(article.Summary, q.Keyword, q.Highlight),
			Author: searchAuthorBrief(article.AuthorID, article.Author), Tags: searchTagBriefs(tagMap[article.ID]), Views: article.Views,
		})
	}
	return results, nil
}

func (s *SearchService) skillSearchResults(ctx context.Context, skills []model.Skill, q SearchQuery) ([]SearchResult, error) {
	ids := make([]uint, 0, len(skills))
	for _, skill := range skills {
		ids = append(ids, skill.ID)
	}
	tagMap, err := s.searchRepo.TagsForSkills(ctx, ids)
	if err != nil {
		return nil, err
	}
	results := make([]SearchResult, 0, len(skills))
	for _, skill := range skills {
		results = append(results, SearchResult{
			ID: skill.ID, Type: "skill",
			Title: searchText(skill.Name, q.Keyword, q.Highlight), Summary: searchText(skill.Description, q.Keyword, q.Highlight),
			Author: searchAuthorBrief(skill.AuthorID, skill.Author), Tags: searchTagBriefs(tagMap[skill.ID]), Views: skill.Views,
		})
	}
	return results, nil
}

func (s *SearchService) mcpServerSearchResults(ctx context.Context, servers []model.McpServer, q SearchQuery) ([]SearchResult, error) {
	ids := make([]uint, 0, len(servers))
	for _, server := range servers {
		ids = append(ids, server.ID)
	}
	tagMap, err := s.searchRepo.TagsForMcpServers(ctx, ids)
	if err != nil {
		return nil, err
	}
	results := make([]SearchResult, 0, len(servers))
	for _, server := range servers {
		results = append(results, SearchResult{
			ID: server.ID, Type: "mcp_server",
			Title: searchText(server.Name, q.Keyword, q.Highlight), Summary: searchText(server.Description, q.Keyword, q.Highlight),
			Author: searchAuthorBrief(server.AuthorID, server.Author), Tags: searchTagBriefs(tagMap[server.ID]), Views: server.Views,
		})
	}
	return results, nil
}

func searchAuthorBrief(authorID uint, author *model.User) AuthorBrief {
	brief := AuthorBrief{ID: authorID}
	if author != nil {
		brief = AuthorBrief{ID: author.ID, Nickname: author.Nickname, AvatarURL: author.AvatarURL}
	}
	return brief
}

func searchTagBriefs(tags []model.Tag) []TagBrief {
	briefs := make([]TagBrief, 0, len(tags))
	for _, tag := range tags {
		briefs = append(briefs, TagBrief{ID: tag.ID, Name: tag.Name})
	}
	return briefs
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
