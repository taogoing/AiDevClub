package service

import (
	"context"

	"aidevclub/internal/model"
	"aidevclub/internal/repo"
)

type RankingService struct {
	articleRepo *repo.ArticleRepo
	skillRepo   *repo.SkillRepo
	mcpRepo     *repo.McpServerRepo
}

func NewRankingService(articleRepo *repo.ArticleRepo, skillRepo *repo.SkillRepo, mcpRepo *repo.McpServerRepo) *RankingService {
	return &RankingService{
		articleRepo: articleRepo,
		skillRepo:   skillRepo,
		mcpRepo:     mcpRepo,
	}
}

func (s *RankingService) ListArticleHot(ctx context.Context, page, pageSize int) ([]ArticleSummary, error) {
	list, _, err := s.articleRepo.List(ctx, repo.ArticleQuery{
		Page:     page,
		PageSize: pageSize,
		Sort:     "hot",
	})
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return []ArticleSummary{}, nil
	}

	ids := make([]uint, 0, len(list))
	for _, a := range list {
		ids = append(ids, a.ID)
	}
	tagMap, err := s.articleRepo.TagsForArticles(ctx, ids)
	if err != nil {
		return nil, err
	}

	result := make([]ArticleSummary, 0, len(list))
	for _, a := range list {
		result = append(result, rankingArticleSummary(a, tagMap[a.ID]))
	}
	return result, nil
}

func (s *RankingService) ListSkillHot(ctx context.Context, page, pageSize int) ([]SkillSummary, error) {
	list, _, err := s.skillRepo.List(ctx, repo.SkillQuery{
		Page:     page,
		PageSize: pageSize,
		Sort:     "hot",
	})
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return []SkillSummary{}, nil
	}

	ids := make([]uint, 0, len(list))
	for _, sk := range list {
		ids = append(ids, sk.ID)
	}
	tagMap, err := s.skillRepo.TagsForSkills(ctx, ids)
	if err != nil {
		return nil, err
	}

	result := make([]SkillSummary, 0, len(list))
	for _, sk := range list {
		result = append(result, rankingSkillSummary(sk, tagMap[sk.ID]))
	}
	return result, nil
}

func (s *RankingService) ListMcpServerHot(ctx context.Context, page, pageSize int) ([]McpServerSummary, error) {
	list, _, err := s.mcpRepo.List(ctx, repo.McpServerQuery{
		Page:     page,
		PageSize: pageSize,
		Sort:     "hot",
	})
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return []McpServerSummary{}, nil
	}

	ids := make([]uint, 0, len(list))
	for _, sv := range list {
		ids = append(ids, sv.ID)
	}
	tagMap, err := s.mcpRepo.TagsForMcpServers(ctx, ids)
	if err != nil {
		return nil, err
	}

	result := make([]McpServerSummary, 0, len(list))
	for _, sv := range list {
		result = append(result, rankingMcpServerSummary(sv, tagMap[sv.ID]))
	}
	return result, nil
}

func rankingArticleSummary(article model.Article, tags []model.Tag) ArticleSummary {
	summary := ArticleSummary{
		ID: article.ID, Title: article.Title, Summary: article.Summary,
		Tags:  rankingTagBriefs(tags),
		Views: article.Views, LikesCount: article.LikesCount,
		FavoritesCount: article.FavoritesCount, CommentsCount: article.CommentsCount,
		Status: string(article.Status), PublishedAt: article.PublishedAt, Pinned: article.Pinned,
		Author: AuthorBrief{ID: article.AuthorID},
	}
	if article.Author != nil {
		summary.Author = AuthorBrief{ID: article.Author.ID, Nickname: article.Author.Nickname, AvatarURL: article.Author.AvatarURL}
	}
	return summary
}

func rankingSkillSummary(skill model.Skill, tags []model.Tag) SkillSummary {
	summary := SkillSummary{
		ID: skill.ID, Name: skill.Name, Description: skill.Description,
		RepoURL: skill.RepoURL, Tags: rankingTagBriefs(tags),
		Views:      skill.Views,
		LikesCount: skill.LikesCount, FavoritesCount: skill.FavoritesCount,
		CommentsCount: skill.CommentsCount, Status: string(skill.Status),
		PublishedAt: skill.PublishedAt, Author: AuthorBrief{ID: skill.AuthorID},
	}
	if skill.Author != nil {
		summary.Author = AuthorBrief{ID: skill.Author.ID, Nickname: skill.Author.Nickname, AvatarURL: skill.Author.AvatarURL}
	}
	return summary
}

func rankingMcpServerSummary(server model.McpServer, tags []model.Tag) McpServerSummary {
	summary := McpServerSummary{
		ID: server.ID, Name: server.Name, Description: server.Description,
		RepoURL: server.RepoURL, Tags: rankingTagBriefs(tags),
		Views:      server.Views,
		LikesCount: server.LikesCount, FavoritesCount: server.FavoritesCount,
		CommentsCount: server.CommentsCount, Status: string(server.Status),
		PublishedAt: server.PublishedAt, Author: AuthorBrief{ID: server.AuthorID},
	}
	if server.Author != nil {
		summary.Author = AuthorBrief{ID: server.Author.ID, Nickname: server.Author.Nickname, AvatarURL: server.Author.AvatarURL}
	}
	return summary
}

func rankingTagBriefs(tags []model.Tag) []TagBrief {
	briefs := make([]TagBrief, 0, len(tags))
	for _, tag := range tags {
		briefs = append(briefs, TagBrief{ID: tag.ID, Name: tag.Name})
	}
	return briefs
}
