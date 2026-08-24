package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"aidevclub/internal/platform"
	"aidevclub/internal/service"
)

type SearchHandler struct {
	svc *service.SearchService
}

func NewSearchHandler(svc *service.SearchService) *SearchHandler {
	return &SearchHandler{svc: svc}
}

func (h *SearchHandler) Search(c *gin.Context) {
	keyword := c.Query("q")
	if keyword == "" {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "搜索关键词不能为空")
		return
	}

	searchType := c.Query("type")

	var tagID, categoryID *uint
	if tagIDStr := c.Query("tag_id"); tagIDStr != "" {
		if id, err := strconv.ParseUint(tagIDStr, 10, 64); err == nil {
			id := uint(id)
			tagID = &id
		}
	}

	if categoryIDStr := c.Query("category_id"); categoryIDStr != "" {
		if id, err := strconv.ParseUint(categoryIDStr, 10, 64); err == nil {
			id := uint(id)
			categoryID = &id
		}
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	result, err := h.svc.Search(c.Request.Context(), keyword, searchType, tagID, categoryID, page, pageSize)
	if err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}

	platform.OK(c, result)
}
