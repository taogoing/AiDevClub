package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"aidevclub/internal/model"
	"aidevclub/internal/service"
)

type fakeProfileReader struct {
	user   *model.User
	err    error
	calls  int
	userID uint
}

func (f *fakeProfileReader) Get(_ context.Context, userID uint) (*model.User, error) {
	f.calls++
	f.userID = userID
	if f.err != nil {
		return nil, f.err
	}
	return f.user, nil
}

type fakeOwnedArticleReader struct {
	result   *service.ArticleListResult
	err      error
	calls    int
	actorID  uint
	status   string
	page     int
	pageSize int
}

func (f *fakeOwnedArticleReader) ListOwned(_ context.Context, actorID uint, status string, page, pageSize int) (*service.ArticleListResult, error) {
	f.calls++
	f.actorID, f.status, f.page, f.pageSize = actorID, status, page, pageSize
	if f.err != nil {
		return nil, f.err
	}
	return f.result, nil
}

type fakeOwnedSkillReader struct {
	result   *service.SkillListResult
	err      error
	calls    int
	actorID  uint
	status   string
	page     int
	pageSize int
}

func (f *fakeOwnedSkillReader) ListOwned(_ context.Context, actorID uint, status string, page, pageSize int) (*service.SkillListResult, error) {
	f.calls++
	f.actorID, f.status, f.page, f.pageSize = actorID, status, page, pageSize
	if f.err != nil {
		return nil, f.err
	}
	return f.result, nil
}

type fakeOwnedMCPServerReader struct {
	result   *service.McpServerListResult
	err      error
	calls    int
	actorID  uint
	status   string
	page     int
	pageSize int
}

func (f *fakeOwnedMCPServerReader) ListOwned(_ context.Context, actorID uint, status string, page, pageSize int) (*service.McpServerListResult, error) {
	f.calls++
	f.actorID, f.status, f.page, f.pageSize = actorID, status, page, pageSize
	if f.err != nil {
		return nil, f.err
	}
	return f.result, nil
}

type fakeAccountNotificationReader struct {
	result           *service.NotificationListResult
	err              error
	calls            int
	actorID          uint
	notificationType string
	unreadOnly       bool
	page             int
	pageSize         int
	markReadCalls    int
	markAllReadCalls int
}

func (f *fakeAccountNotificationReader) List(_ context.Context, actorID uint, notificationType string, unreadOnly bool, page, pageSize int) (*service.NotificationListResult, error) {
	f.calls++
	f.actorID, f.notificationType, f.unreadOnly, f.page, f.pageSize = actorID, notificationType, unreadOnly, page, pageSize
	if f.err != nil {
		return nil, f.err
	}
	return f.result, nil
}

func (f *fakeAccountNotificationReader) MarkRead(context.Context, uint, uint) error {
	f.markReadCalls++
	return nil
}

func (f *fakeAccountNotificationReader) MarkAllRead(context.Context, uint) error {
	f.markAllReadCalls++
	return nil
}

func accountTestDependencies(profile *fakeProfileReader) AccountDependencies {
	return AccountDependencies{
		Profile:       profile,
		Articles:      &fakeOwnedArticleReader{},
		Skills:        &fakeOwnedSkillReader{},
		MCPServers:    &fakeOwnedMCPServerReader{},
		Notifications: &fakeAccountNotificationReader{},
	}
}

func newAccountTestServer(actor Actor, deps AccountDependencies) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "aidevclub-account-test", Version: "test"}, nil)
	RegisterAccountTools(server, deps, actor, testPublicBaseURL)
	return server
}

func schemaContainsProperty(value any, name string) bool {
	switch node := value.(type) {
	case map[string]any:
		if properties, ok := node["properties"].(map[string]any); ok {
			if _, exists := properties[name]; exists {
				return true
			}
		}
		for _, child := range node {
			if schemaContainsProperty(child, name) {
				return true
			}
		}
	case []any:
		for _, child := range node {
			if schemaContainsProperty(child, name) {
				return true
			}
		}
	}
	return false
}

func TestActorFromContextReturnsStoredActorAndDefaultsToAnonymous(t *testing.T) {
	actor := Actor{UserID: 42, Authenticated: true}
	ctx := context.WithValue(context.Background(), actorContextKey{}, actor)
	if got := ActorFromContext(ctx); got != actor {
		t.Fatalf("ActorFromContext() = %#v, want %#v", got, actor)
	}
	if got := ActorFromContext(context.Background()); got != (Actor{}) {
		t.Fatalf("ActorFromContext(background) = %#v, want anonymous actor", got)
	}
}

func TestAccountToolsRegisterExactlyThreeReadOnlyToolsWithoutUserID(t *testing.T) {
	profile := &fakeProfileReader{user: &model.User{ID: 42}}
	tools := listPublicTools(t, newAccountTestServer(Actor{UserID: 42, Authenticated: true}, accountTestDependencies(profile)))
	want := []string{"get_my_profile", "list_my_content", "list_my_notifications"}
	if len(tools) != len(want) {
		t.Fatalf("tool count = %d, want %d", len(tools), len(want))
	}
	for index, tool := range tools {
		if tool.Name != want[index] {
			t.Fatalf("tool[%d].Name = %q, want %q", index, tool.Name, want[index])
		}
		if tool.InputSchema == nil || tool.OutputSchema == nil {
			t.Fatalf("tool %q is missing an explicit SDK schema", tool.Name)
		}
		if tool.Annotations == nil || !tool.Annotations.ReadOnlyHint || !tool.Annotations.IdempotentHint {
			t.Fatalf("tool %q annotations = %#v, want read-only and idempotent", tool.Name, tool.Annotations)
		}
		schema := toolInputSchema(t, tools, tool.Name)
		if schemaContainsProperty(schema, "user_id") {
			t.Fatalf("tool %q unexpectedly advertises user_id: %#v", tool.Name, schema)
		}
	}

	contentSchema := toolInputSchema(t, tools, "list_my_content")
	assertSchemaProperty(t, contentSchema, "content_type", map[string]any{
		"enum": []any{"article", "skill", "mcp_server"},
	})
	assertSchemaProperty(t, contentSchema, "page", map[string]any{"minimum": float64(1), "default": float64(1)})
	assertSchemaProperty(t, contentSchema, "page_size", map[string]any{
		"minimum": float64(1), "maximum": float64(20), "default": float64(10),
	})

	notificationSchema := toolInputSchema(t, tools, "list_my_notifications")
	assertSchemaProperty(t, notificationSchema, "unread_only", map[string]any{"default": false})
	assertSchemaProperty(t, notificationSchema, "page", map[string]any{"minimum": float64(1), "default": float64(1)})
	assertSchemaProperty(t, notificationSchema, "page_size", map[string]any{
		"minimum": float64(1), "maximum": float64(50), "default": float64(20),
	})
}

func TestGetMyProfileRejectsUnauthenticatedActorWithoutReading(t *testing.T) {
	profile := &fakeProfileReader{user: &model.User{ID: 42}}
	result := callToolResult(t, newAccountTestServer(Actor{}, accountTestDependencies(profile)), "get_my_profile", map[string]any{})
	assertToolErrorCode(t, result, errorCodeNotAuthenticated)
	if profile.calls != 0 {
		t.Fatalf("profile calls = %d, want 0", profile.calls)
	}
}

func TestGetMyProfileBindsActorAndOmitsSensitiveFields(t *testing.T) {
	profile := &fakeProfileReader{user: &model.User{
		ID: 42, Email: "private@example.test", PasswordHash: "secret", Nickname: "Ada",
		AvatarURL: "/uploads/ada.png", Bio: "Builder", Role: model.UserRoleAdmin,
	}}
	output := callTool[map[string]any](t, newAccountTestServer(
		Actor{UserID: 42, Authenticated: true}, accountTestDependencies(profile),
	), "get_my_profile", map[string]any{})
	if profile.calls != 1 || profile.userID != 42 {
		t.Fatalf("profile read = (%d calls, user %d), want (1, 42)", profile.calls, profile.userID)
	}
	if output["id"] != float64(42) || output["nickname"] != "Ada" || output["avatar_url"] != testPublicBaseURL+"/uploads/ada.png" || output["bio"] != "Builder" || output["role"] != "admin" {
		t.Fatalf("profile output = %#v", output)
	}
	payload, err := json.Marshal(output)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"email", "password", "authenticated"} {
		if strings.Contains(string(payload), forbidden) {
			t.Fatalf("profile output leaks %q: %s", forbidden, payload)
		}
	}
}

func TestGetMyProfileMapsDeletedActorAndDatabaseFailureToStableErrors(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		wantCode  string
		forbidden string
	}{
		{name: "deleted", err: service.ErrUserNotFound, wantCode: errorCodeNotFound},
		{name: "database", err: errors.New("dial tcp mysql.internal:3306"), wantCode: errorCodeInternal, forbidden: "mysql.internal"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			profile := &fakeProfileReader{err: test.err}
			result := callToolResult(t, newAccountTestServer(
				Actor{UserID: 42, Authenticated: true}, accountTestDependencies(profile),
			), "get_my_profile", map[string]any{})
			assertToolErrorCode(t, result, test.wantCode)
			if test.forbidden != "" && strings.Contains(toolText(result), test.forbidden) {
				t.Fatalf("error leaked database detail: %q", toolText(result))
			}
		})
	}
}

func TestListMyContentBindsActorSelectsOwnedReaderAndRetainsStatus(t *testing.T) {
	actor := Actor{UserID: 42, Authenticated: true}
	profile := &fakeProfileReader{user: &model.User{ID: 42}}
	tests := []struct {
		name        string
		contentType string
		status      string
		section     string
		configure   func(*AccountDependencies)
		assertRead  func(*testing.T, AccountDependencies)
	}{
		{
			name: "article", contentType: "article", status: "draft", section: "articles",
			configure: func(deps *AccountDependencies) {
				deps.Articles = &fakeOwnedArticleReader{result: &service.ArticleListResult{
					List:  []service.ArticleSummary{{ID: 1, Title: "Draft", Status: "draft", Author: service.AuthorBrief{ID: 42}, Tags: []service.TagBrief{}}},
					Total: 1, Page: 2, PageSize: 7,
				}}
			},
			assertRead: func(t *testing.T, deps AccountDependencies) {
				got := deps.Articles.(*fakeOwnedArticleReader)
				if got.calls != 1 || got.actorID != 42 || got.status != "draft" || got.page != 2 || got.pageSize != 7 {
					t.Fatalf("article read = %#v", got)
				}
				if deps.Skills.(*fakeOwnedSkillReader).calls != 0 || deps.MCPServers.(*fakeOwnedMCPServerReader).calls != 0 {
					t.Fatal("non-selected owned readers were called")
				}
			},
		},
		{
			name: "skill", contentType: "skill", status: "rejected", section: "skills",
			configure: func(deps *AccountDependencies) {
				deps.Skills = &fakeOwnedSkillReader{result: &service.SkillListResult{
					List:  []service.SkillSummary{{ID: 2, Name: "Rejected", Status: "rejected", Author: service.AuthorBrief{ID: 42}, Tags: []service.TagBrief{}}},
					Total: 1, Page: 2, PageSize: 7,
				}}
			},
			assertRead: func(t *testing.T, deps AccountDependencies) {
				got := deps.Skills.(*fakeOwnedSkillReader)
				if got.calls != 1 || got.actorID != 42 || got.status != "rejected" || got.page != 2 || got.pageSize != 7 {
					t.Fatalf("skill read = %#v", got)
				}
				if deps.Articles.(*fakeOwnedArticleReader).calls != 0 || deps.MCPServers.(*fakeOwnedMCPServerReader).calls != 0 {
					t.Fatal("non-selected owned readers were called")
				}
			},
		},
		{
			name: "mcp server", contentType: "mcp_server", status: "archived", section: "mcp_servers",
			configure: func(deps *AccountDependencies) {
				deps.MCPServers = &fakeOwnedMCPServerReader{result: &service.McpServerListResult{
					List:  []service.McpServerSummary{{ID: 3, Name: "Archived", Status: "archived", Author: service.AuthorBrief{ID: 42}, Tags: []service.TagBrief{}}},
					Total: 1, Page: 2, PageSize: 7,
				}}
			},
			assertRead: func(t *testing.T, deps AccountDependencies) {
				got := deps.MCPServers.(*fakeOwnedMCPServerReader)
				if got.calls != 1 || got.actorID != 42 || got.status != "archived" || got.page != 2 || got.pageSize != 7 {
					t.Fatalf("MCP server read = %#v", got)
				}
				if deps.Articles.(*fakeOwnedArticleReader).calls != 0 || deps.Skills.(*fakeOwnedSkillReader).calls != 0 {
					t.Fatal("non-selected owned readers were called")
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			deps := accountTestDependencies(profile)
			test.configure(&deps)
			output := callTool[map[string]any](t, newAccountTestServer(actor, deps), "list_my_content", map[string]any{
				"content_type": test.contentType, "status": test.status, "page": 2, "page_size": 7,
			})
			test.assertRead(t, deps)
			if output["content_type"] != test.contentType || output["total"] != float64(1) || output["page"] != float64(2) || output["page_size"] != float64(7) {
				t.Fatalf("page output = %#v", output)
			}
			items, ok := output[test.section].([]any)
			if !ok || len(items) != 1 {
				t.Fatalf("%s = %#v, want one item", test.section, output[test.section])
			}
			item, ok := items[0].(map[string]any)
			if !ok || item["type"] != test.contentType || item["status"] != test.status {
				t.Fatalf("owned item = %#v", items[0])
			}
		})
	}
}

func TestAccountListSummariesDistinguishCurrentPageFromTotal(t *testing.T) {
	actor := Actor{UserID: 42, Authenticated: true}
	profile := &fakeProfileReader{user: &model.User{ID: 42}}

	contentDeps := accountTestDependencies(profile)
	contentDeps.Articles = &fakeOwnedArticleReader{result: &service.ArticleListResult{
		List:  []service.ArticleSummary{{ID: 1, Author: service.AuthorBrief{ID: 42}, Tags: []service.TagBrief{}}},
		Total: 5, Page: 2, PageSize: 1,
	}}
	contentResult := callToolResult(t, newAccountTestServer(actor, contentDeps), "list_my_content", map[string]any{
		"content_type": "article", "page": 2, "page_size": 1,
	})
	if contentResult.IsError {
		t.Fatalf("list_my_content returned tool error: %s", toolText(contentResult))
	}
	contentSummary := toolText(contentResult)
	if !strings.Contains(contentSummary, "returned 1 result(s)") || !strings.Contains(contentSummary, "5 total") {
		t.Fatalf("content summary = %q, want current-page returned count 1 and total 5", contentSummary)
	}

	notificationDeps := accountTestDependencies(profile)
	notificationDeps.Notifications = &fakeAccountNotificationReader{result: &service.NotificationListResult{
		List:  []service.NotificationItem{{ID: 1}},
		Total: 4, Page: 3, PageSize: 1,
	}}
	notificationResult := callToolResult(t, newAccountTestServer(actor, notificationDeps), "list_my_notifications", map[string]any{
		"page": 3, "page_size": 1,
	})
	if notificationResult.IsError {
		t.Fatalf("list_my_notifications returned tool error: %s", toolText(notificationResult))
	}
	notificationSummary := toolText(notificationResult)
	if !strings.Contains(notificationSummary, "returned 1 result(s)") || !strings.Contains(notificationSummary, "4 total") {
		t.Fatalf("notification summary = %q, want current-page returned count 1 and total 4", notificationSummary)
	}
}

func TestListMyContentRejectsInvalidInputBeforeOwnedReads(t *testing.T) {
	tests := []map[string]any{
		{},
		{"content_type": "video"},
		{"content_type": "article", "status": "pending_review"},
		{"content_type": "skill", "status": "unknown"},
		{"content_type": "article", "page": -1},
		{"content_type": "article", "page_size": 21},
	}
	for _, arguments := range tests {
		profile := &fakeProfileReader{user: &model.User{ID: 42}}
		deps := accountTestDependencies(profile)
		result := callToolResult(t, newAccountTestServer(Actor{UserID: 42, Authenticated: true}, deps), "list_my_content", arguments)
		assertToolErrorCode(t, result, errorCodeInvalidArgument)
		if deps.Articles.(*fakeOwnedArticleReader).calls != 0 || deps.Skills.(*fakeOwnedSkillReader).calls != 0 || deps.MCPServers.(*fakeOwnedMCPServerReader).calls != 0 {
			t.Fatalf("owned reader called for invalid arguments %#v", arguments)
		}
	}
}

func TestListMyContentAndNotificationsRecheckDeletedActorBeforeDomainRead(t *testing.T) {
	tests := []struct {
		name string
		tool string
		args map[string]any
	}{
		{name: "content", tool: "list_my_content", args: map[string]any{"content_type": "article"}},
		{name: "notifications", tool: "list_my_notifications", args: map[string]any{}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			profile := &fakeProfileReader{err: service.ErrUserNotFound}
			deps := accountTestDependencies(profile)
			result := callToolResult(t, newAccountTestServer(Actor{UserID: 42, Authenticated: true}, deps), test.tool, test.args)
			assertToolErrorCode(t, result, errorCodeNotFound)
			if deps.Articles.(*fakeOwnedArticleReader).calls != 0 || deps.Notifications.(*fakeAccountNotificationReader).calls != 0 {
				t.Fatal("domain reader called after actor deletion")
			}
		})
	}
}

func TestListMyNotificationsBindsActorFiltersAndNeverMutates(t *testing.T) {
	createdAt := time.Date(2026, 8, 25, 10, 11, 12, 0, time.UTC)
	profile := &fakeProfileReader{user: &model.User{ID: 7}}
	notifications := &fakeAccountNotificationReader{result: &service.NotificationListResult{
		List: []service.NotificationItem{{
			ID: 9, Type: string(model.NotifTypeResourceApproved), Title: "Approved", Content: "Ready",
			ResourceType: "skill", ResourceID: 5, Actor: service.AuthorBrief{ID: 8, Nickname: "Reviewer", AvatarURL: "/reviewer.png"},
			IsRead: false, CreatedAt: createdAt,
		}},
		Total: 1, Page: 3, PageSize: 6,
	}}
	deps := accountTestDependencies(profile)
	deps.Notifications = notifications
	output := callTool[map[string]any](t, newAccountTestServer(Actor{UserID: 7, Authenticated: true}, deps), "list_my_notifications", map[string]any{
		"type": string(model.NotifTypeResourceApproved), "unread_only": true, "page": 3, "page_size": 6,
	})
	if profile.calls != 1 || profile.userID != 7 {
		t.Fatalf("profile read = (%d calls, user %d), want (1, 7)", profile.calls, profile.userID)
	}
	if notifications.calls != 1 || notifications.actorID != 7 || notifications.notificationType != string(model.NotifTypeResourceApproved) || !notifications.unreadOnly || notifications.page != 3 || notifications.pageSize != 6 {
		t.Fatalf("notification read = %#v", notifications)
	}
	if notifications.markReadCalls != 0 || notifications.markAllReadCalls != 0 {
		t.Fatalf("notification mutator calls = (%d, %d), want zero", notifications.markReadCalls, notifications.markAllReadCalls)
	}
	items, ok := output["notifications"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("notifications = %#v, want one item", output["notifications"])
	}
	item := items[0].(map[string]any)
	if item["created_at"] != "2026-08-25T10:11:12Z" || item["is_read"] != false {
		t.Fatalf("notification output = %#v", item)
	}
	actor := item["actor"].(map[string]any)
	if actor["avatar_url"] != testPublicBaseURL+"/reviewer.png" {
		t.Fatalf("notification actor = %#v", actor)
	}
}

func TestListMyNotificationsRejectsInvalidArgumentsWithoutReading(t *testing.T) {
	tests := []map[string]any{
		{"type": "unknown"},
		{"page": -1},
		{"page_size": 51},
	}
	for _, arguments := range tests {
		profile := &fakeProfileReader{user: &model.User{ID: 7}}
		deps := accountTestDependencies(profile)
		result := callToolResult(t, newAccountTestServer(Actor{UserID: 7, Authenticated: true}, deps), "list_my_notifications", arguments)
		assertToolErrorCode(t, result, errorCodeInvalidArgument)
		if deps.Notifications.(*fakeAccountNotificationReader).calls != 0 {
			t.Fatalf("notification reader called for invalid arguments %#v", arguments)
		}
	}
}

func TestListMyContentMapsReaderFailureWithoutLeakingDatabaseDetails(t *testing.T) {
	profile := &fakeProfileReader{user: &model.User{ID: 42}}
	deps := accountTestDependencies(profile)
	deps.Articles = &fakeOwnedArticleReader{err: errors.New("SELECT * FROM articles: connection refused")}
	result := callToolResult(t, newAccountTestServer(Actor{UserID: 42, Authenticated: true}, deps), "list_my_content", map[string]any{
		"content_type": "article",
	})
	assertToolErrorCode(t, result, errorCodeInternal)
	if strings.Contains(toolText(result), "SELECT") || strings.Contains(toolText(result), "connection refused") {
		t.Fatalf("error leaked database detail: %q", toolText(result))
	}
}
