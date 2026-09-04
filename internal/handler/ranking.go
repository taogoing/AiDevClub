package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"aidevclub/internal/platform"
	"aidevclub/internal/service"
)

type RankingHandler struct {
	rankingSvc *service.RankingService
}

func NewRankingHandler(rankingSvc *service.RankingService) *RankingHandler {
	return &RankingHandler{
		rankingSvc: rankingSvc,
	}
}

func (h *RankingHandler) GetArticleRanking(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "5"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 5
	}
	if pageSize > 50 {
		pageSize = 50
	}

	briefs, total, err := h.rankingSvc.GetArticleHotBriefs(c.Request.Context(), page, pageSize)
	if err != nil {
		platform.Fail(c, http.StatusInternalServerError, platform.CodeInternalError, err.Error())
		return
	}
	platform.OK(c, gin.H{
		"articles":  briefs,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}
