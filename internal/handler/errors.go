package handler

import (
	"errors"
	"net/http"

	"aidevclub/internal/platform"
)

func errStatus(err error) int {
	var be *platform.BizError
	if errors.As(err, &be) {
		return be.Status
	}
	return http.StatusInternalServerError
}

func errCode(err error) int {
	var be *platform.BizError
	if errors.As(err, &be) {
		return be.Code
	}
	return 50000
}
