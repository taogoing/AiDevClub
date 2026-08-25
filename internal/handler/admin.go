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
	r.GET("/users", h.ListUsers)
	r.PUT("/users/:id/role", h.UpdateUserRole)
	r.GET("/articles", h.ListArticles)
	r.GET("/articles/:id", h.GetArticle)
	r.PUT("/articles/:id/hide", h.HideArticle)
	r.PUT("/articles/:id/unhide", h.UnhideArticle)
	r.GET("/comments", h.ListArticleComments)
	r.PUT("/comments/:id/hide", h.HideComment)
	r.PUT("/comments/:id/unhide", h.UnhideComment)
	r.GET("/resource-comments", h.ListResourceComments)
	r.PUT("/resource-comments/:id/hide", h.HideResourceComment)
	r.PUT("/resource-comments/:id/unhide", h.UnhideResourceComment)
	r.GET("/skills", h.ListSkills)
	r.GET("/skills/:id", h.GetSkill)
	r.PUT("/skills/:id/hide", h.HideSkill)
	r.PUT("/skills/:id/unhide", h.UnhideSkill)
	r.PUT("/skills/:id/review", h.ReviewSkill)
	r.GET("/mcp-servers", h.ListMCPServers)
	r.GET("/mcp-servers/:id", h.GetMCPServer)
	r.PUT("/mcp-servers/:id/hide", h.HideMcpServer)
	r.PUT("/mcp-servers/:id/unhide", h.UnhideMcpServer)
	r.PUT("/mcp-servers/:id/review", h.ReviewMcpServer)
	r.POST("/announcements", h.CreateAnnouncement)
	r.GET("/announcements", h.ListAnnouncements)
	r.GET("/reports", h.ListReports)
	r.GET("/reports/:id", h.GetReport)
	r.PUT("/reports/:id/resolve", h.ResolveReport)
	r.GET("/logs", h.ListLogs)
}

func (h *AdminHandler) ListUsers(c *gin.Context) {
	keyword := c.Query("keyword")
	role := model.UserRole(c.Query("role"))
	page := queryInt(c, "page", 1)
	pageSize := queryInt(c, "page_size", 20)
	res, err := h.adminSvc.ListUsers(c.Request.Context(), c.GetUint("user_id"), keyword, role, page, pageSize)
	if err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, res)
}

func (h *AdminHandler) UpdateUserRole(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "参数错误")
		return
	}
	var in struct {
		Role model.UserRole `json:"role"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "参数错误")
		return
	}
	if in.Role != model.UserRoleUser && in.Role != model.UserRoleAdmin {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "无效的角色")
		return
	}
	if err := h.adminSvc.UpdateUserRole(c.Request.Context(), c.GetUint("user_id"), id, in.Role); err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, nil)
}

func (h *AdminHandler) ListArticles(c *gin.Context) {
	keyword := c.Query("keyword")
	visibility := c.Query("visibility")
	var authorID *uint
	if aid := c.Query("author_id"); aid != "" {
		id := parseUint(aid)
		if id > 0 {
			authorID = &id
		}
	}
	page := queryInt(c, "page", 1)
	pageSize := queryInt(c, "page_size", 20)
	res, err := h.adminSvc.ListArticles(c.Request.Context(), service.AdminArticleQuery{
		Keyword:    keyword,
		Visibility: visibility,
		AuthorID:   authorID,
		Page:       page,
		PageSize:   pageSize,
	})
	if err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, res)
}

func (h *AdminHandler) GetArticle(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "参数错误")
		return
	}
	res, err := h.adminSvc.GetArticle(c.Request.Context(), id)
	if err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, res)
}

func (h *AdminHandler) ListArticleComments(c *gin.Context) {
	keyword := c.Query("keyword")
	visibility := c.Query("visibility")
	page := queryInt(c, "page", 1)
	pageSize := queryInt(c, "page_size", 20)
	res, err := h.adminSvc.ListArticleComments(c.Request.Context(), service.AdminCommentQuery{
		Keyword:    keyword,
		Visibility: visibility,
		Page:       page,
		PageSize:   pageSize,
	})
	if err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, res)
}

func (h *AdminHandler) HideComment(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "参数错误")
		return
	}
	if err := h.adminSvc.HideComment(c.Request.Context(), c.GetUint("user_id"), id); err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, nil)
}

func (h *AdminHandler) UnhideComment(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "参数错误")
		return
	}
	if err := h.adminSvc.UnhideComment(c.Request.Context(), c.GetUint("user_id"), id); err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, nil)
}

func (h *AdminHandler) ListResourceComments(c *gin.Context) {
	keyword := c.Query("keyword")
	visibility := c.Query("visibility")
	resourceType := c.Query("resource_type")
	page := queryInt(c, "page", 1)
	pageSize := queryInt(c, "page_size", 20)
	res, err := h.adminSvc.ListResourceComments(c.Request.Context(), service.AdminResourceCommentQuery{
		Keyword:      keyword,
		Visibility:   visibility,
		ResourceType: resourceType,
		Page:         page,
		PageSize:     pageSize,
	})
	if err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, res)
}

func (h *AdminHandler) HideResourceComment(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "参数错误")
		return
	}
	if err := h.adminSvc.HideResourceComment(c.Request.Context(), c.GetUint("user_id"), id); err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, nil)
}

func (h *AdminHandler) UnhideResourceComment(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "参数错误")
		return
	}
	if err := h.adminSvc.UnhideResourceComment(c.Request.Context(), c.GetUint("user_id"), id); err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, nil)
}

func (h *AdminHandler) ListSkills(c *gin.Context) {
	keyword := c.Query("keyword")
	status := c.Query("status")
	var authorID *uint
	if aid := c.Query("author_id"); aid != "" {
		id := parseUint(aid)
		if id > 0 {
			authorID = &id
		}
	}
	var tagID *uint
	if tid := c.Query("tag_id"); tid != "" {
		id := parseUint(tid)
		if id > 0 {
			tagID = &id
		}
	}
	page := queryInt(c, "page", 1)
	pageSize := queryInt(c, "page_size", 20)
	res, err := h.adminSvc.ListSkills(c.Request.Context(), service.AdminResourceQuery{
		Keyword:  keyword,
		Status:   status,
		AuthorID: authorID,
		TagID:    tagID,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, res)
}

func (h *AdminHandler) GetSkill(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "参数错误")
		return
	}
	res, err := h.adminSvc.GetSkill(c.Request.Context(), id)
	if err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, res)
}

func (h *AdminHandler) ListMCPServers(c *gin.Context) {
	keyword := c.Query("keyword")
	status := c.Query("status")
	var authorID *uint
	if aid := c.Query("author_id"); aid != "" {
		id := parseUint(aid)
		if id > 0 {
			authorID = &id
		}
	}
	var tagID *uint
	if tid := c.Query("tag_id"); tid != "" {
		id := parseUint(tid)
		if id > 0 {
			tagID = &id
		}
	}
	page := queryInt(c, "page", 1)
	pageSize := queryInt(c, "page_size", 20)
	res, err := h.adminSvc.ListMCPServers(c.Request.Context(), service.AdminResourceQuery{
		Keyword:  keyword,
		Status:   status,
		AuthorID: authorID,
		TagID:    tagID,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, res)
}

func (h *AdminHandler) GetMCPServer(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "参数错误")
		return
	}
	res, err := h.adminSvc.GetMCPServer(c.Request.Context(), id)
	if err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, res)
}

func (h *AdminHandler) GetReport(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "参数错误")
		return
	}
	res, err := h.reportSvc.AdminGet(c.Request.Context(), id)
	if err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, res)
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
