package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"aidevclub/internal/platform"
	"aidevclub/internal/service"
)

type TagHandler struct{ svc *service.TagService }

func NewTagHandler(svc *service.TagService) *TagHandler { return &TagHandler{svc: svc} }

func (h *TagHandler) List(c *gin.Context) {
	prefix := c.Query("prefix")
	hot := c.Query("hot") == "1"
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	contentType := c.Query("type")

	list, err := h.svc.List(c.Request.Context(), prefix, hot, limit, contentType)
	if err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	out := make([]gin.H, 0, len(list))
	for _, t := range list {
		out = append(out, gin.H{"id": t.ID, "name": t.Name, "description": t.Description, "usage_count": t.UsageCount})
	}
	platform.OK(c, out)
}
