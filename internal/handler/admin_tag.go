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
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Enabled     *bool   `json:"enabled"`
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

	updates := make(map[string]interface{})
	if req.Name != nil {
		if *req.Name == "" {
			platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "标签名称不能为空")
			return
		}
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}

	if len(updates) == 0 {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "没有要更新的字段")
		return
	}

	if err := h.svc.AdminUpdate(c.Request.Context(), id, updates); err != nil {
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
