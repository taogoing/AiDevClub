package mcpserver

import (
	"encoding/json"
	"errors"
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
