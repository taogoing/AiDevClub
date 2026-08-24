package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"aidevclub/internal/platform"
	"aidevclub/internal/service"
)

type AdminTagHandler struct {
	svc *service.TagService
}

func NewAdminTagHandler(svc *service.TagService) *AdminTagHandler {
	return &AdminTagHandler{svc: svc}
}

type createTagRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

func (h *AdminTagHandler) Create(c *gin.Context) {
	var req createTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "参数错误")
		return
	}

	tag, err := h.svc.AdminCreate(c.Request.Context(), req.Name, req.Description)
	if err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}

	platform.OK(c, tag)
}

type updateTagRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

func (h *AdminTagHandler) Update(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "无效的标签 ID")
		return
	}

	var req updateTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "参数错误")
		return
	}

	if err := h.svc.AdminUpdate(c.Request.Context(), id, req.Name, req.Description); err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}

	platform.OK(c, nil)
}

func (h *AdminTagHandler) Enable(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "无效的标签 ID")
		return
	}

	if err := h.svc.Enable(c.Request.Context(), id); err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}

	platform.OK(c, nil)
}

func (h *AdminTagHandler) Disable(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "无效的标签 ID")
		return
	}

	if err := h.svc.Disable(c.Request.Context(), id); err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}

	platform.OK(c, nil)
}

func (h *AdminTagHandler) List(c *gin.Context) {
	keyword := c.Query("keyword")
	status := c.DefaultQuery("status", "all")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	tags, total, err := h.svc.AdminList(c.Request.Context(), keyword, status, page, pageSize)
	if err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}

	platform.OK(c, gin.H{
		"items":     tags,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}
