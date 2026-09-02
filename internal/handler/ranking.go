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

	articles, total, err := h.rankingSvc.ListArticleHot(c.Request.Context(), page, pageSize)
	if err != nil {
		platform.Fail(c, http.StatusInternalServerError, platform.CodeInternalError, err.Error())
		return
	}
	platform.OK(c, gin.H{
		"articles":  articles,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}
