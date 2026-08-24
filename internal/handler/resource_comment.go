package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"aidevclub/internal/platform"
	"aidevclub/internal/service"
)

type ResourceCommentHandler struct{ svc *service.ResourceCommentService }

func NewResourceCommentHandler(svc *service.ResourceCommentService) *ResourceCommentHandler {
	return &ResourceCommentHandler{svc: svc}
}

func (h *ResourceCommentHandler) List(c *gin.Context) {
	resourceType := c.GetString("resource_type")
	id, err := parseUintParam(c, "id")
	if err != nil {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "参数错误")
		return
	}
	list, err := h.svc.List(c.Request.Context(), resourceType, id)
	if err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, list)
}

func (h *ResourceCommentHandler) Create(c *gin.Context) {
	resourceType := c.GetString("resource_type")
	id, err := parseUintParam(c, "id")
	if err != nil {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "参数错误")
		return
	}
	var in struct {
		Content  string `json:"content"`
		ParentID *uint  `json:"parent_id"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "参数错误")
		return
	}
	comment, err := h.svc.Create(c.Request.Context(), c.GetUint("user_id"), resourceType, id, in.Content, in.ParentID)
	if err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, gin.H{"id": comment.ID})
}

func (h *ResourceCommentHandler) Delete(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "参数错误")
		return
	}
	if err := h.svc.Delete(c.Request.Context(), c.GetUint("user_id"), id); err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, nil)
}

func (h *ResourceCommentHandler) Like(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "参数错误")
		return
	}
	liked, count, err := h.svc.ToggleLike(c.Request.Context(), c.GetUint("user_id"), id)
	if err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, gin.H{"liked": liked, "likes_count": count})
}
