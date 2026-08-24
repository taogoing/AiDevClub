package handler

import (
	"github.com/gin-gonic/gin"

	"aidevclub/internal/platform"
	"aidevclub/internal/service"
)

type CategoryHandler struct{ svc *service.CategoryService }

func NewCategoryHandler(svc *service.CategoryService) *CategoryHandler { return &CategoryHandler{svc: svc} }

func (h *CategoryHandler) List(c *gin.Context) {
	list, err := h.svc.List(c.Request.Context())
	if err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	out := make([]gin.H, 0, len(list))
	for _, cat := range list {
		out = append(out, gin.H{"id": cat.ID, "name": cat.Name, "slug": cat.Slug})
	}
	platform.OK(c, out)
}
