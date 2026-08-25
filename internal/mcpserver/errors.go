package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	errorCodeInvalidArgument        = "invalid_argument"
	errorCodeContentNotFound        = "content_not_found"
	errorCodeResultTooLarge         = "result_too_large"
	errorCodeTemporarilyUnavailable = "temporarily_unavailable"
	errorCodeInternal               = "internal_error"
	errorCodeNotAuthenticated       = "not_authenticated"
	errorCodeNotFound               = "not_found"
)

type toolError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *toolError) Error() string {
	payload, _ := json.Marshal(e)
	return string(payload)
}

func invalidArgument(message string) error {
	return &toolError{Code: errorCodeInvalidArgument, Message: message}
}

func internalError() error {
	return &toolError{Code: errorCodeInternal, Message: "The request could not be completed."}
}

func temporarilyUnavailable() error {
	return &toolError{Code: errorCodeTemporarilyUnavailable, Message: "Content is temporarily unavailable."}
}

func notAuthenticated() error {
	return &toolError{Code: errorCodeNotAuthenticated, Message: "Authentication is required."}
}

func accountNotFound() error {
	return &toolError{Code: errorCodeNotFound, Message: "The authenticated account was not found."}
}

func contentReadError(err error, notFound error) error {
	if errors.Is(err, notFound) {
		return &toolError{Code: errorCodeContentNotFound, Message: "The requested content was not found."}
	}
	return internalError()
}

func stablePublicToolSchemaErrors(next mcp.MethodHandler) mcp.MethodHandler {
	return stableToolSchemaErrors(isPublicTool, next)
}

func stableAccountToolSchemaErrors(next mcp.MethodHandler) mcp.MethodHandler {
	return stableToolSchemaErrors(isAccountTool, next)
}

func stableToolSchemaErrors(matches func(string) bool, next mcp.MethodHandler) mcp.MethodHandler {
	return func(ctx context.Context, method string, request mcp.Request) (mcp.Result, error) {
		result, err := next(ctx, method, request)
		if err != nil {
			return result, err
		}
		call, ok := request.(*mcp.CallToolRequest)
		if !ok || call.Params == nil || !matches(call.Params.Name) {
			return result, nil
		}
		toolResult, ok := result.(*mcp.CallToolResult)
		if !ok || toolResult.GetError() == nil || !strings.HasPrefix(toolResult.GetError().Error(), `validating "arguments":`) {
			return result, nil
		}
		stable := invalidArgument("The tool arguments do not match the advertised schema.")
		toolResult.Content = nil
		toolResult.SetError(stable)
		return toolResult, nil
	}
}

func isAccountTool(name string) bool {
	switch name {
	case "get_my_profile", "list_my_content", "list_my_notifications":
		return true
	default:
		return false
	}
}

func isPublicTool(name string) bool {
	switch name {
	case "search_content", "browse_content", "get_article", "get_skill", "get_mcp_server", "list_taxonomy":
		return true
	default:
		return false
	}
}
