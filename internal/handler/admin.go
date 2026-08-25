package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"aidevclub/internal/model"
	"aidevclub/internal/platform"
	"aidevclub/internal/service"
)

type AdminHandler struct {
	adminSvc    *service.AdminService
	reportSvc   *service.ReportService
	adminLogSvc *service.AdminLogService
}

func NewAdminHandler(adminSvc *service.AdminService, reportSvc *service.ReportService, adminLogSvc *service.AdminLogService) *AdminHandler {
	return &AdminHandler{adminSvc: adminSvc, reportSvc: reportSvc, adminLogSvc: adminLogSvc}
}

func (h *AdminHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/dashboard", h.Dashboard)
	r.PUT("/articles/:id/hide", h.HideArticle)
	r.PUT("/articles/:id/unhide", h.UnhideArticle)
	r.PUT("/skills/:id/hide", h.HideSkill)
	r.PUT("/skills/:id/unhide", h.UnhideSkill)
	r.PUT("/skills/:id/review", h.ReviewSkill)
	r.PUT("/mcp-servers/:id/hide", h.HideMcpServer)
	r.PUT("/mcp-servers/:id/unhide", h.UnhideMcpServer)
	r.PUT("/mcp-servers/:id/review", h.ReviewMcpServer)
	r.POST("/announcements", h.CreateAnnouncement)
	r.GET("/announcements", h.ListAnnouncements)
	r.GET("/reports", h.ListReports)
	r.PUT("/reports/:id/resolve", h.ResolveReport)
	r.GET("/logs", h.ListLogs)
}

func (h *AdminHandler) Dashboard(c *gin.Context) {
	data, err := h.adminSvc.Dashboard(c.Request.Context())
	if err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, data)
}

func (h *AdminHandler) HideArticle(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "参数错误")
		return
	}
	if err := h.adminSvc.HideArticle(c.Request.Context(), c.GetUint("user_id"), id); err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, nil)
}

func (h *AdminHandler) UnhideArticle(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "参数错误")
		return
	}
	if err := h.adminSvc.UnhideArticle(c.Request.Context(), c.GetUint("user_id"), id); err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, nil)
}

func (h *AdminHandler) HideSkill(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "参数错误")
		return
	}
	if err := h.adminSvc.HideSkill(c.Request.Context(), c.GetUint("user_id"), id); err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, nil)
}

func (h *AdminHandler) UnhideSkill(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "参数错误")
		return
	}
	if err := h.adminSvc.UnhideSkill(c.Request.Context(), c.GetUint("user_id"), id); err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, nil)
}

func (h *AdminHandler) HideMcpServer(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "参数错误")
		return
	}
	if err := h.adminSvc.HideMcpServer(c.Request.Context(), c.GetUint("user_id"), id); err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, nil)
}

func (h *AdminHandler) UnhideMcpServer(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "参数错误")
		return
	}
	if err := h.adminSvc.UnhideMcpServer(c.Request.Context(), c.GetUint("user_id"), id); err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, nil)
}

func (h *AdminHandler) ReviewSkill(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "参数错误")
		return
	}
	var in struct {
		Approved bool   `json:"approved"`
		Reason   string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "参数错误")
		return
	}
	if err := h.adminSvc.ReviewSkill(c.Request.Context(), c.GetUint("user_id"), id, in.Approved, in.Reason); err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, nil)
}

func (h *AdminHandler) ReviewMcpServer(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "参数错误")
		return
	}
	var in struct {
		Approved bool   `json:"approved"`
		Reason   string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "参数错误")
		return
	}
	if err := h.adminSvc.ReviewMcpServer(c.Request.Context(), c.GetUint("user_id"), id, in.Approved, in.Reason); err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, nil)
}

func (h *AdminHandler) CreateAnnouncement(c *gin.Context) {
	var in struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "参数错误")
		return
	}
	ann, err := h.adminSvc.CreateAnnouncement(c.Request.Context(), c.GetUint("user_id"), in.Title, in.Content)
	if err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, gin.H{"id": ann.ID})
}

func (h *AdminHandler) ListAnnouncements(c *gin.Context) {
	page := queryInt(c, "page", 1)
	pageSize := queryInt(c, "page_size", 20)
	res, err := h.adminSvc.ListAnnouncements(c.Request.Context(), page, pageSize)
	if err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, res)
}

func (h *AdminHandler) ListReports(c *gin.Context) {
	page := queryInt(c, "page", 1)
	pageSize := queryInt(c, "page_size", 20)
	status := model.ReportStatus(c.Query("status"))
	res, err := h.reportSvc.List(c.Request.Context(), status, page, pageSize)
	if err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, res)
}

func (h *AdminHandler) ResolveReport(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "参数错误")
		return
	}
	var in struct {
		Action string `json:"action"`
		Result string `json:"result"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "参数错误")
		return
	}
	if err := h.reportSvc.Resolve(c.Request.Context(), c.GetUint("user_id"), id, in.Action, in.Result); err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, nil)
}

func (h *AdminHandler) ListLogs(c *gin.Context) {
	page := queryInt(c, "page", 1)
	pageSize := queryInt(c, "page_size", 20)
	action := model.AdminLogAction(c.Query("action"))
	res, err := h.adminLogSvc.List(c.Request.Context(), action, page, pageSize)
	if err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, res)
}
