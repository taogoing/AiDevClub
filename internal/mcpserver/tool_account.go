package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"aidevclub/internal/model"
	"aidevclub/internal/service"
)

type getMyProfileInput struct{}

type getMyProfileOutput struct {
	ID        uint   `json:"id"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url,omitempty"`
	Bio       string `json:"bio,omitempty"`
	Role      string `json:"role"`
}

type ListMyContentInput struct {
	ContentType string `json:"content_type" jsonschema:"Content type: article, skill, or mcp_server."`
	Status      string `json:"status,omitempty" jsonschema:"Optional status filter for the selected content type."`
	Page        int    `json:"page,omitempty" jsonschema:"Page number, starting at 1."`
	PageSize    int    `json:"page_size,omitempty" jsonschema:"Results per page."`
}

type myContentOutput struct {
	ID          uint         `json:"id"`
	Type        string       `json:"type"`
	Title       string       `json:"title"`
	Summary     string       `json:"summary"`
	URL         string       `json:"url"`
	Author      AuthorOutput `json:"author"`
	Tags        []TagOutput  `json:"tags"`
	Status      string       `json:"status"`
	Views       int          `json:"views"`
	PublishedAt string       `json:"published_at,omitempty"`
}

type listMyContentOutput struct {
	ContentType string            `json:"content_type"`
	Articles    []myContentOutput `json:"articles"`
	Skills      []myContentOutput `json:"skills"`
	MCPServers  []myContentOutput `json:"mcp_servers"`
	Total       int64             `json:"total"`
	PageInfo
}

type listMyNotificationsInput struct {
	Type       string `json:"type,omitempty" jsonschema:"Optional notification type filter."`
	UnreadOnly bool   `json:"unread_only,omitempty" jsonschema:"Return only unread notifications."`
	Page       int    `json:"page,omitempty" jsonschema:"Page number, starting at 1."`
	PageSize   int    `json:"page_size,omitempty" jsonschema:"Results per page."`
}

type notificationOutput struct {
	ID           uint         `json:"id"`
	Type         string       `json:"type"`
	Title        string       `json:"title"`
	Content      string       `json:"content"`
	ResourceType string       `json:"resource_type,omitempty"`
	ResourceID   uint         `json:"resource_id,omitempty"`
	Actor        AuthorOutput `json:"actor"`
	IsRead       bool         `json:"is_read"`
	CreatedAt    string       `json:"created_at"`
}

type listMyNotificationsOutput struct {
	Notifications []notificationOutput `json:"notifications"`
	Total         int64                `json:"total"`
	PageInfo
}

func RegisterAccountTools(server *mcp.Server, deps AccountDependencies, actor Actor, publicBaseURL string) {
	server.AddReceivingMiddleware(stableAccountToolSchemaErrors)
	annotations := &mcp.ToolAnnotations{ReadOnlyHint: true, IdempotentHint: true}
	mcp.AddTool(server, &mcp.Tool{
		Name: "get_my_profile", Description: "Read the authenticated AIDevClub account profile.", Annotations: annotations,
		InputSchema: getMyProfileInputSchema(),
	}, getMyProfile(deps.Profile, actor, publicBaseURL))
	mcp.AddTool(server, &mcp.Tool{
		Name: "list_my_content", Description: "List content owned by the authenticated AIDevClub account.", Annotations: annotations,
		InputSchema: listMyContentInputSchema(),
	}, listMyContent(deps, actor, publicBaseURL))
	mcp.AddTool(server, &mcp.Tool{
		Name: "list_my_notifications", Description: "List notifications without changing their read state.", Annotations: annotations,
		InputSchema: listMyNotificationsInputSchema(),
	}, listMyNotifications(deps, actor, publicBaseURL))
}

func getMyProfileInputSchema() *jsonschema.Schema {
	schema := mustInputSchema[getMyProfileInput]()
	disallowAdditionalProperties(schema)
	return schema
}

func listMyContentInputSchema() *jsonschema.Schema {
	schema := mustInputSchema[ListMyContentInput]()
	schema.Required = []string{"content_type"}
	schema.Properties["content_type"].Enum = []any{"article", "skill", "mcp_server"}
	schema.Properties["page"].Minimum = jsonschema.Ptr(float64(1))
	schema.Properties["page"].Default = json.RawMessage(`1`)
	schema.Properties["page_size"].Minimum = jsonschema.Ptr(float64(1))
	schema.Properties["page_size"].Maximum = jsonschema.Ptr(float64(20))
	schema.Properties["page_size"].Default = json.RawMessage(`10`)
	schema.AllOf = append(schema.AllOf, &jsonschema.Schema{
		If: &jsonschema.Schema{
			Required: []string{"content_type"},
			Properties: map[string]*jsonschema.Schema{
				"content_type": {Enum: []any{"article"}},
			},
		},
		Then: &jsonschema.Schema{Properties: map[string]*jsonschema.Schema{
			"status": {Enum: []any{"draft", "published"}},
		}},
		Else: &jsonschema.Schema{Properties: map[string]*jsonschema.Schema{
			"status": {Enum: []any{"draft", "pending_review", "published", "rejected", "archived"}},
		}},
	})
	disallowAdditionalProperties(schema)
	return schema
}

func listMyNotificationsInputSchema() *jsonschema.Schema {
	schema := mustInputSchema[listMyNotificationsInput]()
	schema.Properties["type"].Enum = notificationTypeValues()
	schema.Properties["unread_only"].Default = json.RawMessage(`false`)
	schema.Properties["page"].Minimum = jsonschema.Ptr(float64(1))
	schema.Properties["page"].Default = json.RawMessage(`1`)
	schema.Properties["page_size"].Minimum = jsonschema.Ptr(float64(1))
	schema.Properties["page_size"].Maximum = jsonschema.Ptr(float64(50))
	schema.Properties["page_size"].Default = json.RawMessage(`20`)
	disallowAdditionalProperties(schema)
	return schema
}

func disallowAdditionalProperties(schema *jsonschema.Schema) {
	schema.AdditionalProperties = &jsonschema.Schema{Not: &jsonschema.Schema{}}
}

func getMyProfile(profileReader ProfileReader, actor Actor, publicBaseURL string) mcp.ToolHandlerFor[getMyProfileInput, getMyProfileOutput] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, _ getMyProfileInput) (*mcp.CallToolResult, getMyProfileOutput, error) {
		profile, err := readActorProfile(ctx, profileReader, actor)
		if err != nil {
			return nil, getMyProfileOutput{}, err
		}
		output := getMyProfileOutput{
			ID: profile.ID, Nickname: profile.Nickname, AvatarURL: absolutePublicURL(publicBaseURL, profile.AvatarURL),
			Bio: profile.Bio, Role: string(profile.Role),
		}
		return summaryResult(fmt.Sprintf("Profile %d returned.", profile.ID)), output, nil
	}
}

func listMyContent(deps AccountDependencies, actor Actor, publicBaseURL string) mcp.ToolHandlerFor[ListMyContentInput, listMyContentOutput] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input ListMyContentInput) (*mcp.CallToolResult, listMyContentOutput, error) {
		page, pageSize, err := normalizeAccountPage(input.Page, input.PageSize, 10, 20)
		if err != nil {
			return nil, listMyContentOutput{}, err
		}
		if err := validateOwnedStatus(input.ContentType, input.Status); err != nil {
			return nil, listMyContentOutput{}, err
		}
		if _, err := readActorProfile(ctx, deps.Profile, actor); err != nil {
			return nil, listMyContentOutput{}, err
		}

		output := listMyContentOutput{
			ContentType: input.ContentType,
			Articles:    []myContentOutput{},
			Skills:      []myContentOutput{},
			MCPServers:  []myContentOutput{},
			PageInfo:    PageInfo{Page: page, PageSize: pageSize},
		}
		switch input.ContentType {
		case "article":
			result, err := deps.Articles.ListOwned(ctx, actor.UserID, input.Status, page, pageSize)
			if err != nil || result == nil {
				return nil, listMyContentOutput{}, internalError()
			}
			output.Articles = ownedArticleOutputs(result.List, publicBaseURL)
			output.Total, output.Page, output.PageSize = result.Total, result.Page, result.PageSize
		case "skill":
			result, err := deps.Skills.ListOwned(ctx, actor.UserID, input.Status, page, pageSize)
			if err != nil || result == nil {
				return nil, listMyContentOutput{}, internalError()
			}
			output.Skills = ownedSkillOutputs(result.List, publicBaseURL)
			output.Total, output.Page, output.PageSize = result.Total, result.Page, result.PageSize
		case "mcp_server":
			result, err := deps.MCPServers.ListOwned(ctx, actor.UserID, input.Status, page, pageSize)
			if err != nil || result == nil {
				return nil, listMyContentOutput{}, internalError()
			}
			output.MCPServers = ownedMCPServerOutputs(result.List, publicBaseURL)
			output.Total, output.Page, output.PageSize = result.Total, result.Page, result.PageSize
		default:
			return nil, listMyContentOutput{}, invalidArgument("content_type must be article, skill, or mcp_server")
		}
		returned := len(output.Articles) + len(output.Skills) + len(output.MCPServers)
		return summaryResult(fmt.Sprintf("Account content returned %d result(s) on this page (%d total).", returned, output.Total)), output, nil
	}
}

func listMyNotifications(deps AccountDependencies, actor Actor, publicBaseURL string) mcp.ToolHandlerFor[listMyNotificationsInput, listMyNotificationsOutput] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input listMyNotificationsInput) (*mcp.CallToolResult, listMyNotificationsOutput, error) {
		page, pageSize, err := normalizeAccountPage(input.Page, input.PageSize, 20, 50)
		if err != nil {
			return nil, listMyNotificationsOutput{}, err
		}
		if input.Type != "" && !validNotificationType(input.Type) {
			return nil, listMyNotificationsOutput{}, invalidArgument("type is not a supported notification type")
		}
		if _, err := readActorProfile(ctx, deps.Profile, actor); err != nil {
			return nil, listMyNotificationsOutput{}, err
		}
		result, err := deps.Notifications.List(ctx, actor.UserID, input.Type, input.UnreadOnly, page, pageSize)
		if err != nil || result == nil {
			return nil, listMyNotificationsOutput{}, internalError()
		}
		output := listMyNotificationsOutput{
			Notifications: notificationOutputs(result.List, publicBaseURL),
			Total:         result.Total,
			PageInfo:      PageInfo{Page: result.Page, PageSize: result.PageSize},
		}
		return summaryResult(fmt.Sprintf("Notifications returned %d result(s) on this page (%d total).", len(output.Notifications), output.Total)), output, nil
	}
}

func readActorProfile(ctx context.Context, reader ProfileReader, actor Actor) (*model.User, error) {
	if !actor.Authenticated || actor.UserID == 0 {
		return nil, notAuthenticated()
	}
	if reader == nil {
		return nil, internalError()
	}
	profile, err := reader.Get(ctx, actor.UserID)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			return nil, accountNotFound()
		}
		return nil, internalError()
	}
	if profile == nil || profile.ID != actor.UserID {
		return nil, internalError()
	}
	return profile, nil
}

func normalizeAccountPage(page, pageSize, defaultPageSize, maxPageSize int) (int, int, error) {
	if page < 0 || pageSize < 0 {
		return 0, 0, invalidArgument("page and page_size must be positive")
	}
	if page == 0 {
		page = 1
	}
	if pageSize == 0 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		return 0, 0, invalidArgument(fmt.Sprintf("page_size must not exceed %d", maxPageSize))
	}
	return page, pageSize, nil
}

func validateOwnedStatus(contentType, status string) error {
	switch contentType {
	case "article":
		if status == "" || status == string(model.ArticleStatusDraft) || status == string(model.ArticleStatusPublished) {
			return nil
		}
	case "skill", "mcp_server":
		switch model.ResourceStatus(status) {
		case "", model.ResourceStatusDraft, model.ResourceStatusPendingReview, model.ResourceStatusPublished, model.ResourceStatusRejected, model.ResourceStatusArchived:
			return nil
		}
	default:
		return invalidArgument("content_type must be article, skill, or mcp_server")
	}
	return invalidArgument("status is not valid for the selected content_type")
}

func ownedArticleOutputs(summaries []service.ArticleSummary, publicBaseURL string) []myContentOutput {
	output := make([]myContentOutput, 0, len(summaries))
	for _, summary := range summaries {
		output = append(output, myContentOutput{
			ID: summary.ID, Type: "article", Title: summary.Title, Summary: summary.Summary,
			URL: contentPageURL(publicBaseURL, "article", summary.ID), Author: authorOutput(summary.Author, publicBaseURL),
			Tags: tagOutputs(summary.Tags), Status: summary.Status, Views: summary.Views, PublishedAt: publishedAtOutput(summary.PublishedAt),
		})
	}
	return output
}

func ownedSkillOutputs(summaries []service.SkillSummary, publicBaseURL string) []myContentOutput {
	output := make([]myContentOutput, 0, len(summaries))
	for _, summary := range summaries {
		output = append(output, myContentOutput{
			ID: summary.ID, Type: "skill", Title: summary.Name, Summary: summary.Description,
			URL: contentPageURL(publicBaseURL, "skill", summary.ID), Author: authorOutput(summary.Author, publicBaseURL),
			Tags: tagOutputs(summary.Tags), Status: summary.Status, Views: summary.Views,
			PublishedAt: publishedAtOutput(summary.PublishedAt),
		})
	}
	return output
}

func ownedMCPServerOutputs(summaries []service.McpServerSummary, publicBaseURL string) []myContentOutput {
	output := make([]myContentOutput, 0, len(summaries))
	for _, summary := range summaries {
		output = append(output, myContentOutput{
			ID: summary.ID, Type: "mcp_server", Title: summary.Name, Summary: summary.Description,
			URL: contentPageURL(publicBaseURL, "mcp_server", summary.ID), Author: authorOutput(summary.Author, publicBaseURL),
			Tags: tagOutputs(summary.Tags), Status: summary.Status, Views: summary.Views,
			PublishedAt: publishedAtOutput(summary.PublishedAt),
		})
	}
	return output
}

func notificationOutputs(items []service.NotificationItem, publicBaseURL string) []notificationOutput {
	output := make([]notificationOutput, 0, len(items))
	for _, item := range items {
		output = append(output, notificationOutput{
			ID: item.ID, Type: item.Type, Title: item.Title, Content: item.Content,
			ResourceType: item.ResourceType, ResourceID: item.ResourceID, Actor: authorOutput(item.Actor, publicBaseURL),
			IsRead: item.IsRead, CreatedAt: item.CreatedAt.Format(time.RFC3339),
		})
	}
	return output
}

func notificationTypeValues() []any {
	return []any{
		string(model.NotifTypeCommentArticle),
		string(model.NotifTypeReplyComment),
		string(model.NotifTypeLikeArticle),
		string(model.NotifTypeLikeSkill),
		string(model.NotifTypeLikeMcpServer),
		string(model.NotifTypeLikeComment),
		string(model.NotifTypeLikeResourceComment),
		string(model.NotifTypeResourceApproved),
		string(model.NotifTypeResourceRejected),
		string(model.NotifTypeReportResolved),
		string(model.NotifTypeAnnouncement),
	}
}

func validNotificationType(value string) bool {
	for _, candidate := range notificationTypeValues() {
		if value == candidate {
			return true
		}
	}
	return false
}
