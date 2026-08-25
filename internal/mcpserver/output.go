package mcpserver

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"aidevclub/internal/service"
)

type PageInfo struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

type AuthorOutput struct {
	ID        uint   `json:"id"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url,omitempty"`
}

type TagOutput struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	UsageCount  int    `json:"usage_count,omitempty"`
}

type contentWindow struct {
	Text       string
	HasMore    bool
	NextOffset int
}

func unicodeWindow(content string, offset, limit int) contentWindow {
	runes := []rune(content)
	start := offset
	if start > len(runes) {
		start = len(runes)
	}
	end := start + limit
	if end > len(runes) {
		end = len(runes)
	}
	return contentWindow{
		Text:       string(runes[start:end]),
		HasMore:    end < len(runes),
		NextOffset: end,
	}
}

func authorOutput(author service.AuthorBrief, publicBaseURL string) AuthorOutput {
	return AuthorOutput{
		ID: author.ID, Nickname: author.Nickname,
		AvatarURL: absolutePublicURL(publicBaseURL, author.AvatarURL),
	}
}

func tagOutputs(tags []service.TagBrief) []TagOutput {
	output := make([]TagOutput, 0, len(tags))
	for _, tag := range tags {
		output = append(output, TagOutput{ID: tag.ID, Name: tag.Name})
	}
	return output
}

func publishedAtOutput(publishedAt *time.Time) string {
	if publishedAt == nil {
		return ""
	}
	return publishedAt.Format(time.RFC3339)
}

func contentPageURL(publicBaseURL, contentType string, id uint) string {
	route := contentType
	if contentType == "mcp_server" {
		route = "mcps"
	} else {
		route += "s"
	}
	return strings.TrimRight(publicBaseURL, "/") + fmt.Sprintf("/%s/%d", route, id)
}

func absolutePublicURL(publicBaseURL, value string) string {
	if value == "" {
		return ""
	}
	parsed, err := url.Parse(value)
	if err == nil && parsed.IsAbs() {
		return value
	}
	return strings.TrimRight(publicBaseURL, "/") + "/" + strings.TrimLeft(value, "/")
}
