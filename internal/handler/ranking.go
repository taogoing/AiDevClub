package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"aidevclub/internal/platform"
	"aidevclub/internal/service"
)

type RankingHandler struct {
	rankingSvc *service.ContentRankingService
}

func NewRankingHandler(rankingSvc *service.ContentRankingService) *RankingHandler {
	return &RankingHandler{rankingSvc: rankingSvc}
}

func rankingPageParams(c *gin.Context) (int, int) {
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
	return page, pageSize
}

func (h *RankingHandler) GetArticleRanking(c *gin.Context) {
	page, pageSize := rankingPageParams(c)
	briefs, total, err := h.rankingSvc.ListArticles(c.Request.Context(), page, pageSize)
	if err != nil {
		platform.Fail(c, http.StatusInternalServerError, platform.CodeInternalError, err.Error())
		return
	}
	platform.OK(c, gin.H{
		"articles": briefs, "total": total, "page": page, "page_size": pageSize,
	})
}

func (h *RankingHandler) GetSkillRanking(c *gin.Context) {
	page, pageSize := rankingPageParams(c)
	briefs, total, err := h.rankingSvc.ListSkills(c.Request.Context(), page, pageSize)
	if err != nil {
		platform.Fail(c, http.StatusInternalServerError, platform.CodeInternalError, err.Error())
		return
	}
	platform.OK(c, gin.H{
		"skills": briefs, "total": total, "page": page, "page_size": pageSize,
	})
}

func (h *RankingHandler) GetMcpServerRanking(c *gin.Context) {
	page, pageSize := rankingPageParams(c)
	briefs, total, err := h.rankingSvc.ListMcpServers(c.Request.Context(), page, pageSize)
	if err != nil {
		platform.Fail(c, http.StatusInternalServerError, platform.CodeInternalError, err.Error())
		return
	}
	platform.OK(c, gin.H{
		"mcp_servers": briefs, "total": total, "page": page, "page_size": pageSize,
	})
}
