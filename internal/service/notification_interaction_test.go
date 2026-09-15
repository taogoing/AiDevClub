package service

import (
	"context"
	"testing"

	"gorm.io/gorm"

	"aidevclub/internal/model"
	"aidevclub/internal/platform"
	"aidevclub/internal/repo"
	"aidevclub/internal/testutil"
)

func TestFavoriteInteractionsCreateNotificationsBeforeReturning(t *testing.T) {
	db := testutil.NewTestDB(t)
	ctx := context.Background()
	users := repo.NewUserRepo(db)
	author := seedUser(t, db, "favorite-author@notification.test")
	actor := seedUser(t, db, "favorite-actor@notification.test")
	notifications := NewNotificationService(repo.NewNotificationRepo(db), users)
	interactions := repo.NewInteractionRepo(db)
	cfg := &platform.Config{DefaultPageSize: 20, MaxPageSize: 50}

	article := seedPublishedArticle(t, db, author.ID, "favorite article")
	articleService := NewArticleService(repo.NewArticleRepo(db), repo.NewTagRepo(db), interactions, cfg, notifications, nil)
	if favorited, _, err := articleService.ToggleFavorite(ctx, actor.ID, article.ID); err != nil || !favorited {
		t.Fatalf("article favorite = %v, %v", favorited, err)
	}

	skill := seedSkill(t, db, author.ID, "favorite-skill", model.ResourceStatusPublished, false)
	skillService := NewSkillService(repo.NewSkillRepo(db), repo.NewTagRepo(db), interactions, cfg, notifications, nil)
	if favorited, _, err := skillService.ToggleFavorite(ctx, actor.ID, skill.ID); err != nil || !favorited {
		t.Fatalf("skill favorite = %v, %v", favorited, err)
	}

	server := seedMcpServer(t, db, author.ID, "favorite-server", model.ResourceStatusPublished, false)
	serverService := NewMcpServerService(repo.NewMcpServerRepo(db), repo.NewTagRepo(db), interactions, cfg, notifications, nil)
	if favorited, _, err := serverService.ToggleFavorite(ctx, actor.ID, server.ID); err != nil || !favorited {
		t.Fatalf("MCP server favorite = %v, %v", favorited, err)
	}

	wantTypes := map[model.NotifType]bool{
		model.NotifTypeFavoriteArticle:   false,
		model.NotifTypeFavoriteSkill:     false,
		model.NotifTypeFavoriteMcpServer: false,
	}
	var actual []model.Notification
	if err := db.Where("user_id = ?", author.ID).Find(&actual).Error; err != nil {
		t.Fatal(err)
	}
	for _, notification := range actual {
		if _, ok := wantTypes[notification.Type]; ok {
			wantTypes[notification.Type] = true
		}
	}
	for notifType, found := range wantTypes {
		if !found {
			t.Errorf("notification type %q not persisted before interaction returned; rows: %+v", notifType, actual)
		}
	}

	if favorited, _, err := articleService.ToggleFavorite(ctx, actor.ID, article.ID); err != nil || favorited {
		t.Fatalf("article unfavorite = %v, %v", favorited, err)
	}
	var articleFavoriteNotifications int64
	if err := db.Model(&model.Notification{}).Where("type = ? AND resource_id = ?", model.NotifTypeFavoriteArticle, article.ID).Count(&articleFavoriteNotifications).Error; err != nil {
		t.Fatal(err)
	}
	if articleFavoriteNotifications != 1 {
		t.Fatalf("unfavorite created an extra notification: count=%d", articleFavoriteNotifications)
	}

	commentService := NewCommentService(repo.NewCommentRepo(db), repo.NewArticleRepo(db), interactions, users, notifications, nil)
	comment, err := commentService.Create(ctx, actor.ID, article.ID, "synchronous article comment", nil)
	if err != nil {
		t.Fatal(err)
	}
	assertNotificationCount(t, db, author.ID, model.NotifTypeCommentArticle, "article", article.ID, 1)
	commentLiker := seedUser(t, db, "comment-liker@notification.test")
	if liked, _, err := commentService.ToggleLike(ctx, commentLiker.ID, comment.ID); err != nil || !liked {
		t.Fatalf("comment like = %v, %v", liked, err)
	}
	assertNotificationCount(t, db, actor.ID, model.NotifTypeLikeComment, "comment", comment.ID, 1)

	resourceComments := NewResourceCommentService(repo.NewResourceCommentRepo(db), repo.NewSkillRepo(db), repo.NewMcpServerRepo(db), interactions, users, notifications, nil)
	resourceComment, err := resourceComments.Create(ctx, actor.ID, "skill", skill.ID, "synchronous resource comment", nil)
	if err != nil {
		t.Fatal(err)
	}
	assertNotificationCount(t, db, author.ID, model.NotifTypeCommentArticle, "skill", skill.ID, 1)
	if liked, _, err := resourceComments.ToggleLike(ctx, commentLiker.ID, resourceComment.ID); err != nil || !liked {
		t.Fatalf("resource comment like = %v, %v", liked, err)
	}
	assertNotificationCount(t, db, actor.ID, model.NotifTypeLikeResourceComment, "skill", resourceComment.ID, 1)
}

func assertNotificationCount(t *testing.T, db *gorm.DB, userID uint, notifType model.NotifType, resourceType string, resourceID uint, want int64) {
	t.Helper()
	var count int64
	if err := db.Model(&model.Notification{}).Where("user_id = ? AND type = ? AND resource_type = ? AND resource_id = ?", userID, notifType, resourceType, resourceID).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != want {
		t.Fatalf("notifications user=%d type=%s resource=%s/%d = %d, want %d", userID, notifType, resourceType, resourceID, count, want)
	}
}

func TestOutboxModeKeepsBusinessSuccessWhenRabbitMQIsUnavailable(t *testing.T) {
	db := testutil.NewTestDB(t)
	ctx := context.Background()
	users := repo.NewUserRepo(db)
	author := seedUser(t, db, "outbox-author@notification.test")
	actor := seedUser(t, db, "outbox-actor@notification.test")
	outbox := repo.NewNotificationOutboxRepo(db)
	notifications := NewNotificationService(repo.NewNotificationRepo(db), users)
	notifications.ConfigureMode("outbox", outbox)
	article := seedPublishedArticle(t, db, author.ID, "outbox article")
	articleService := NewArticleService(repo.NewArticleRepo(db), repo.NewTagRepo(db), repo.NewInteractionRepo(db), &platform.Config{}, notifications, nil)

	liked, count, err := articleService.ToggleLike(ctx, actor.ID, article.ID)
	if err != nil || !liked || count != 1 {
		t.Fatalf("like with unavailable broker = liked:%v count:%d err:%v", liked, count, err)
	}
	if favorited, _, err := articleService.ToggleFavorite(ctx, actor.ID, article.ID); err != nil || !favorited {
		t.Fatalf("favorite with unavailable broker = %v, %v", favorited, err)
	}
	commentService := NewCommentService(repo.NewCommentRepo(db), repo.NewArticleRepo(db), repo.NewInteractionRepo(db), users, notifications, nil)
	if _, err := commentService.Create(ctx, actor.ID, article.ID, "outbox comment", nil); err != nil {
		t.Fatalf("comment with unavailable broker: %v", err)
	}
	var likeCount, outboxCount, notificationCount int64
	if err := db.Model(&model.ArticleLike{}).Where("article_id = ? AND user_id = ?", article.ID, actor.ID).Count(&likeCount).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.NotificationOutboxEvent{}).Where("status = ?", model.NotificationOutboxPending).Count(&outboxCount).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.Notification{}).Count(&notificationCount).Error; err != nil {
		t.Fatal(err)
	}
	if likeCount != 1 || outboxCount != 3 || notificationCount != 0 {
		t.Fatalf("like=%d pending outbox=%d notifications=%d", likeCount, outboxCount, notificationCount)
	}
}

func TestOutboxInsertFailureRollsBackBusinessInteraction(t *testing.T) {
	db := testutil.NewTestDB(t)
	ctx := context.Background()
	users := repo.NewUserRepo(db)
	author := seedUser(t, db, "outbox-fail-author@notification.test")
	actor := seedUser(t, db, "outbox-fail-actor@notification.test")
	notifications := NewNotificationService(repo.NewNotificationRepo(db), users)
	notifications.ConfigureMode("outbox", repo.NewNotificationOutboxRepo(db))
	article := seedPublishedArticle(t, db, author.ID, "outbox rollback article")
	articleService := NewArticleService(repo.NewArticleRepo(db), repo.NewTagRepo(db), repo.NewInteractionRepo(db), &platform.Config{}, notifications, nil)
	if err := db.Migrator().DropTable(&model.NotificationOutboxEvent{}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := articleService.ToggleLike(ctx, actor.ID, article.ID); err == nil {
		t.Fatal("like with unavailable outbox table succeeded; want rollback error")
	}
	var likeCount int64
	if err := db.Model(&model.ArticleLike{}).Where("article_id = ? AND user_id = ?", article.ID, actor.ID).Count(&likeCount).Error; err != nil {
		t.Fatal(err)
	}
	if likeCount != 0 {
		t.Fatalf("business interaction committed without outbox event: likes=%d", likeCount)
	}
}

func TestSyncNotificationFailureDoesNotFailCommittedLike(t *testing.T) {
	db := testutil.NewTestDB(t)
	ctx := context.Background()
	users := repo.NewUserRepo(db)
	author := seedUser(t, db, "sync-fail-author@notification.test")
	actor := seedUser(t, db, "sync-fail-actor@notification.test")
	notifications := NewNotificationService(repo.NewNotificationRepo(db), users)
	article := seedPublishedArticle(t, db, author.ID, "sync best-effort article")
	articleService := NewArticleService(repo.NewArticleRepo(db), repo.NewTagRepo(db), repo.NewInteractionRepo(db), &platform.Config{}, notifications, nil)
	if err := db.Migrator().DropTable(&model.Notification{}); err != nil {
		t.Fatal(err)
	}
	liked, count, err := articleService.ToggleLike(ctx, actor.ID, article.ID)
	if err != nil || !liked || count != 1 {
		t.Fatalf("like with failed best-effort notification = liked:%v count:%d err:%v", liked, count, err)
	}
	var likes int64
	if err := db.Model(&model.ArticleLike{}).Where("article_id = ? AND user_id = ?", article.ID, actor.ID).Count(&likes).Error; err != nil {
		t.Fatal(err)
	}
	if likes != 1 {
		t.Fatalf("committed likes=%d, want 1", likes)
	}
}
