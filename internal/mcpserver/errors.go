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

func contentReadError(err error, notFound error) error {
	if errors.Is(err, notFound) {
		return &toolError{Code: errorCodeContentNotFound, Message: "The requested content was not found."}
	}
	return internalError()
}

func stablePublicToolSchemaErrors(next mcp.MethodHandler) mcp.MethodHandler {
	return func(ctx context.Context, method string, request mcp.Request) (mcp.Result, error) {
		result, err := next(ctx, method, request)
		if err != nil {
			return result, err
		}
		call, ok := request.(*mcp.CallToolRequest)
		if !ok || call.Params == nil || !isPublicTool(call.Params.Name) {
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

func isPublicTool(name string) bool {
	switch name {
	case "search_content", "browse_content", "get_article", "get_skill", "get_mcp_server", "list_taxonomy":
		return true
	default:
		return false
	}
}
