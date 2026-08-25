package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"aidevclub/internal/model"
	"aidevclub/internal/repo"
	"aidevclub/internal/testutil"
)

func newMCPReadTestDB(t *testing.T) (*gorm.DB, *model.User, *model.Category) {
	t.Helper()
	db := testutil.NewTestDB(t)
	ctx := context.Background()
	users := repo.NewUserRepo(db)
	user := &model.User{Email: "mcp-reader@example.com", PasswordHash: "x", Nickname: "MCP Reader"}
	if err := users.Create(user); err != nil {
		t.Fatal(err)
	}
	category := &model.Category{Name: "Go", Slug: "go", SortOrder: 1}
	if err := db.WithContext(ctx).Create(category).Error; err != nil {
		t.Fatal(err)
	}
	return db, user, category
}

func TestNotificationListUnreadFilterOwnsCountAndPage(t *testing.T) {
	db, user, _ := newMCPReadTestDB(t)
	ctx := context.Background()
	now := time.Now().UTC()
	notifications := []model.Notification{
		{UserID: user.ID, Type: model.NotifTypeLikeArticle, Title: "old unread", Content: "one", CreatedAt: now.Add(-3 * time.Minute)},
		{UserID: user.ID, Type: model.NotifTypeLikeArticle, Title: "new unread", Content: "two", CreatedAt: now.Add(-2 * time.Minute)},
		{UserID: user.ID, Type: model.NotifTypeLikeArticle, Title: "new read", Content: "three", IsRead: true, CreatedAt: now.Add(-time.Minute)},
	}
	if err := db.WithContext(ctx).Create(&notifications).Error; err != nil {
		t.Fatal(err)
	}
	svc := NewNotificationService(repo.NewNotificationRepo(db), repo.NewUserRepo(db))

	first, err := svc.List(ctx, user.ID, "", true, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if first.Total != 2 || len(first.List) != 1 || first.List[0].Title != "new unread" {
		t.Fatalf("first unread page = %+v, want total 2 and newest unread row", first)
	}

	second, err := svc.List(ctx, user.ID, "", true, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	if second.Total != 2 || len(second.List) != 1 || second.List[0].Title != "old unread" {
		t.Fatalf("second unread page = %+v, want total 2 and oldest unread row", second)
	}
}

func TestSearchWithoutHighlightReturnsPlainText(t *testing.T) {
	db, user, category := newMCPReadTestDB(t)
	ctx := context.Background()
	article := &model.Article{
		AuthorID: user.ID, CategoryID: category.ID,
		Title: "Go search", Summary: "A Go summary", Content: "Go search content",
		Status: model.ArticleStatusPublished,
	}
	if err := db.WithContext(ctx).Create(article).Error; err != nil {
		t.Fatal(err)
	}
	search := NewSearchService(repo.NewSearchRepo(db))

	got, err := search.Search(ctx, SearchQuery{Keyword: "Go", Type: "article", Page: 1, PageSize: 10, Highlight: false})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Articles) != 1 {
		t.Fatalf("article search results = %+v, want one article", got)
	}
	if strings.Contains(got.Articles[0].Title, "<mark>") || strings.Contains(got.Articles[0].Summary, "<mark>") {
		t.Fatalf("plain search result contains HTML highlight markup: %+v", got.Articles[0])
	}
}

func TestSearchAllKeepsResultsInSeparateTypeSections(t *testing.T) {
	db, user, category := newMCPReadTestDB(t)
	ctx := context.Background()
	if err := db.WithContext(ctx).Create(&model.Article{
		AuthorID: user.ID, CategoryID: category.ID, Title: "Go article", Summary: "article", Content: "Go article content", Status: model.ArticleStatusPublished,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.WithContext(ctx).Create(&model.Skill{
		AuthorID: user.ID, Name: "Go skill", Description: "skill", Status: model.ResourceStatusPublished,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.WithContext(ctx).Create(&model.McpServer{
		AuthorID: user.ID, Name: "Go server", Description: "server", ToolsJSON: "[]", Status: model.ResourceStatusPublished,
	}).Error; err != nil {
		t.Fatal(err)
	}

	got, err := NewSearchService(repo.NewSearchRepo(db)).Search(ctx, SearchQuery{Keyword: "Go", ContentType: "all", Page: 1, PageSize: 10, Highlight: false})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Articles) != 1 || len(got.Skills) != 1 || len(got.McpServers) != 1 {
		t.Fatalf("all search sections = articles:%+v skills:%+v servers:%+v, want one row in each", got.Articles, got.Skills, got.McpServers)
	}
	if len(got.Items) != 3 || got.Items[0].Type != "article" || got.Items[1].Type != "skill" || got.Items[2].Type != "mcp_server" {
		t.Fatalf("legacy all search items = %+v, want deterministic type-order compatibility projection", got.Items)
	}
}

func TestRankingBatchHydrationPreservesZSetOrderAndDropsUnavailableRows(t *testing.T) {
	db, user, category := newMCPReadTestDB(t)
	ctx := context.Background()
	rdb := testutil.NewTestRedis(t)
	articles := repo.NewArticleRepo(db)
	skills := repo.NewSkillRepo(db)
	servers := repo.NewMcpServerRepo(db)
	ranking := NewRankingService(rdb, articles, skills, servers, 1.5)

	articleFirst := &model.Article{AuthorID: user.ID, CategoryID: category.ID, Title: "article first", Content: "first", Status: model.ArticleStatusPublished}
	articleSecond := &model.Article{AuthorID: user.ID, CategoryID: category.ID, Title: "article second", Content: "second", Status: model.ArticleStatusPublished}
	articleHidden := &model.Article{AuthorID: user.ID, CategoryID: category.ID, Title: "article hidden", Content: "hidden", Status: model.ArticleStatusPublished, Hidden: true}
	articleDeleted := &model.Article{AuthorID: user.ID, CategoryID: category.ID, Title: "article deleted", Content: "deleted", Status: model.ArticleStatusPublished}
	if err := db.WithContext(ctx).Create([]*model.Article{articleFirst, articleSecond, articleHidden, articleDeleted}).Error; err != nil {
		t.Fatal(err)
	}
	if err := articles.Delete(db.WithContext(ctx), articleDeleted.ID); err != nil {
		t.Fatal(err)
	}
	if err := rdb.ZAdd(ctx, "rank:articles:hot",
		redis.Z{Score: 10, Member: articleFirst.ID},
		redis.Z{Score: 20, Member: articleSecond.ID},
		redis.Z{Score: 30, Member: articleHidden.ID},
		redis.Z{Score: 40, Member: articleDeleted.ID},
	).Err(); err != nil {
		t.Fatal(err)
	}
	gotArticles, err := ranking.ListArticleHot(ctx, 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(gotArticles) != 2 || gotArticles[0].ID != articleSecond.ID || gotArticles[1].ID != articleFirst.ID {
		t.Fatalf("article hot summaries = %+v, want Redis order of visible rows only", gotArticles)
	}

	skillFirst := &model.Skill{AuthorID: user.ID, Name: "skill first", Status: model.ResourceStatusPublished}
	skillSecond := &model.Skill{AuthorID: user.ID, Name: "skill second", Status: model.ResourceStatusPublished}
	skillHidden := &model.Skill{AuthorID: user.ID, Name: "skill hidden", Status: model.ResourceStatusPublished, Hidden: true}
	skillDeleted := &model.Skill{AuthorID: user.ID, Name: "skill deleted", Status: model.ResourceStatusPublished}
	if err := db.WithContext(ctx).Create([]*model.Skill{skillFirst, skillSecond, skillHidden, skillDeleted}).Error; err != nil {
		t.Fatal(err)
	}
	if err := skills.Delete(db.WithContext(ctx), skillDeleted.ID); err != nil {
		t.Fatal(err)
	}
	if err := rdb.ZAdd(ctx, "rank:skills:hot",
		redis.Z{Score: 10, Member: skillFirst.ID},
		redis.Z{Score: 20, Member: skillSecond.ID},
		redis.Z{Score: 30, Member: skillHidden.ID},
		redis.Z{Score: 40, Member: skillDeleted.ID},
	).Err(); err != nil {
		t.Fatal(err)
	}
	gotSkills, err := ranking.ListSkillHot(ctx, 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(gotSkills) != 2 || gotSkills[0].ID != skillSecond.ID || gotSkills[1].ID != skillFirst.ID {
		t.Fatalf("skill hot summaries = %+v, want Redis order of visible rows only", gotSkills)
	}

	serverFirst := &model.McpServer{AuthorID: user.ID, Name: "server first", ToolsJSON: "[]", Status: model.ResourceStatusPublished}
	serverSecond := &model.McpServer{AuthorID: user.ID, Name: "server second", ToolsJSON: "[]", Status: model.ResourceStatusPublished}
	serverHidden := &model.McpServer{AuthorID: user.ID, Name: "server hidden", ToolsJSON: "[]", Status: model.ResourceStatusPublished, Hidden: true}
	serverDeleted := &model.McpServer{AuthorID: user.ID, Name: "server deleted", ToolsJSON: "[]", Status: model.ResourceStatusPublished}
	if err := db.WithContext(ctx).Create([]*model.McpServer{serverFirst, serverSecond, serverHidden, serverDeleted}).Error; err != nil {
		t.Fatal(err)
	}
	if err := servers.Delete(db.WithContext(ctx), serverDeleted.ID); err != nil {
		t.Fatal(err)
	}
	if err := rdb.ZAdd(ctx, "rank:mcp_servers:hot",
		redis.Z{Score: 10, Member: serverFirst.ID},
		redis.Z{Score: 20, Member: serverSecond.ID},
		redis.Z{Score: 30, Member: serverHidden.ID},
		redis.Z{Score: 40, Member: serverDeleted.ID},
	).Err(); err != nil {
		t.Fatal(err)
	}
	gotServers, err := ranking.ListMcpServerHot(ctx, 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(gotServers) != 2 || gotServers[0].ID != serverSecond.ID || gotServers[1].ID != serverFirst.ID {
		t.Fatalf("MCP server hot summaries = %+v, want Redis order of visible rows only", gotServers)
	}
}

func TestTaxonomyReadersFilterCategoriesAndDisabledTags(t *testing.T) {
	db, _, category := newMCPReadTestDB(t)
	ctx := context.Background()
	if err := db.WithContext(ctx).Create(&model.Category{Name: "Rust", Slug: "rust", SortOrder: 2}).Error; err != nil {
		t.Fatal(err)
	}
	disabledTag := &model.Tag{Name: "go-hidden"}
	if err := db.WithContext(ctx).Create([]*model.Tag{
		{Name: "golang", Enabled: true},
		disabledTag,
		{Name: "rust", Enabled: true},
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.WithContext(ctx).Model(disabledTag).Update("enabled", false).Error; err != nil {
		t.Fatal(err)
	}

	categories, err := NewCategoryService(repo.NewCategoryRepo(db)).ListForMCP(ctx, "go", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(categories) != 1 || categories[0].ID != category.ID {
		t.Fatalf("filtered categories = %+v, want Go only", categories)
	}
	tags, err := NewTagService(repo.NewTagRepo(db), nil).ListForMCP(ctx, "go", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(tags) != 1 || tags[0].Name != "golang" {
		t.Fatalf("taxonomy tags = %+v, want enabled golang only", tags)
	}
	defaultTags, err := NewTagService(repo.NewTagRepo(db), nil).ListForMCP(ctx, "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(defaultTags) != 2 {
		t.Fatalf("default taxonomy tags = %+v, want both enabled tags", defaultTags)
	}
}
