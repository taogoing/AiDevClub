package platform

const (
	CodeParamError   = 40001
	CodeBizError     = 40002
	CodeStateError   = 40003
	CodeUnauthorized = 40101
	CodeForbidden    = 40301

	CodeUserNotFound       = 40401
	CodeArticleNotFound    = 40402
	CodeCommentNotFound    = 40403
	CodeCategoryNotFound   = 40404
	CodeTagNotFound        = 40405
	CodeSkillNotFound      = 40406
	CodeMcpServerNotFound  = 40407
	CodeResCommentNotFound = 40408
	CodeReportNotFound       = 40409
	CodeNotifNotFound        = 40410
	CodeAnnouncementNotFound = 40411

	CodeEmailExists = 40901

	CodeInternalError = 50000
)

type BizError struct {
	Status  int
	Code    int
	Message string
}

func (e *BizError) Error() string { return e.Message }

func NewBizError(status, code int, message string) *BizError {
	return &BizError{Status: status, Code: code, Message: message}
}
