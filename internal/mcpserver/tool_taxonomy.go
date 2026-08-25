package mcpserver

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type listTaxonomyInput struct {
	Kind    string `json:"kind,omitempty" jsonschema:"Taxonomy kind: all, categories, or tags."`
	Keyword string `json:"keyword,omitempty" jsonschema:"Optional case-insensitive taxonomy keyword."`
	Limit   int    `json:"limit,omitempty" jsonschema:"Maximum results per taxonomy kind, from 1 through 100."`
}

type categoryOutput struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type listTaxonomyOutput struct {
	Kind       string           `json:"kind"`
	Categories []categoryOutput `json:"categories"`
	Tags       []TagOutput      `json:"tags"`
}

func listTaxonomy(deps PublicDependencies, _ string) mcp.ToolHandlerFor[listTaxonomyInput, listTaxonomyOutput] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input listTaxonomyInput) (*mcp.CallToolResult, listTaxonomyOutput, error) {
		kind := strings.TrimSpace(input.Kind)
		if kind == "" {
			kind = "all"
		}
		if kind != "all" && kind != "categories" && kind != "tags" {
			return nil, listTaxonomyOutput{}, invalidArgument("kind must be all, categories, or tags")
		}
		if input.Limit < 0 || input.Limit > 100 {
			return nil, listTaxonomyOutput{}, invalidArgument("limit must be from 1 through 100")
		}
		limit := input.Limit
		if limit == 0 {
			limit = 50
		}
		keyword := strings.TrimSpace(input.Keyword)
		output := listTaxonomyOutput{Kind: kind, Categories: []categoryOutput{}, Tags: []TagOutput{}}
		if kind == "all" || kind == "categories" {
			if deps.Categories == nil {
				return nil, listTaxonomyOutput{}, internalError()
			}
			categories, err := deps.Categories.ListForMCP(ctx, keyword, limit)
			if err != nil {
				return nil, listTaxonomyOutput{}, internalError()
			}
			for _, category := range categories {
				output.Categories = append(output.Categories, categoryOutput{ID: category.ID, Name: category.Name, Slug: category.Slug})
			}
		}
		if kind == "all" || kind == "tags" {
			if deps.Tags == nil {
				return nil, listTaxonomyOutput{}, internalError()
			}
			tags, err := deps.Tags.ListForMCP(ctx, keyword, limit)
			if err != nil {
				return nil, listTaxonomyOutput{}, internalError()
			}
			for _, tag := range tags {
				output.Tags = append(output.Tags, TagOutput{
					ID: tag.ID, Name: tag.Name, Description: tag.Description, UsageCount: tag.UsageCount,
				})
			}
		}
		return summaryResult(fmt.Sprintf("Taxonomy returned %d categories and %d tags.", len(output.Categories), len(output.Tags))), output, nil
	}
}
