package platform

type BizError struct {
	Status  int
	Code    int
	Message string
}

func (e *BizError) Error() string { return e.Message }

func NewBizError(status, code int, message string) *BizError {
	return &BizError{Status: status, Code: code, Message: message}
}
