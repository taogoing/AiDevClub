package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"aidevclub/internal/model"
	"aidevclub/internal/platform"
	"aidevclub/internal/service"
)

type ReportHandler struct{ svc *service.ReportService }

func NewReportHandler(svc *service.ReportService) *ReportHandler {
	return &ReportHandler{svc: svc}
}

func (h *ReportHandler) Create(c *gin.Context) {
	var in struct {
		TargetType  string `json:"target_type"`
		TargetID    uint   `json:"target_id"`
		Reason      string `json:"reason"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "参数错误")
		return
	}
	if in.TargetType == "" || in.TargetID == 0 || in.Reason == "" {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "参数错误")
		return
	}
	reason := model.ReportReason(in.Reason)
	switch reason {
	case model.ReportReasonSpam, model.ReportReasonAbuse, model.ReportReasonCopyright, model.ReportReasonOther:
	default:
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "不支持的举报原因")
		return
	}
	report, err := h.svc.Create(c.Request.Context(), c.GetUint("user_id"), in.TargetType, in.TargetID, reason, in.Description)
	if err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, gin.H{"id": report.ID})
}

func (h *ReportHandler) List(c *gin.Context) {
	page := queryInt(c, "page", 1)
	pageSize := queryInt(c, "page_size", 20)
	res, err := h.svc.ListByReporter(c.Request.Context(), c.GetUint("user_id"), page, pageSize)
	if err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, res)
}
