package handler

import (
	"github.com/gin-gonic/gin"

	"aidevclub/internal/platform"
	"aidevclub/internal/service"
)

type TagHandler struct{ svc *service.TagService }

func NewTagHandler(svc *service.TagService) *TagHandler { return &TagHandler{svc: svc} }

func (h *TagHandler) List(c *gin.Context) {
	keyword := c.Query("keyword")
	hot := c.Query("hot") == "1"
	list, err := h.svc.List(c.Request.Context(), keyword, hot, 50)
	if err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	out := make([]gin.H, 0, len(list))
	for _, t := range list {
		out = append(out, gin.H{"id": t.ID, "name": t.Name, "usage_count": t.UsageCount})
	}
	platform.OK(c, out)
}
