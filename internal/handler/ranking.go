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
	articleSvc *service.ArticleService
	skillSvc   *service.SkillService
	mcpSvc     *service.McpServerService
}

func NewRankingHandler(rankingSvc *service.RankingService, articleSvc *service.ArticleService, skillSvc *service.SkillService, mcpSvc *service.McpServerService) *RankingHandler {
	return &RankingHandler{
		rankingSvc: rankingSvc,
		articleSvc: articleSvc,
		skillSvc:   skillSvc,
		mcpSvc:     mcpSvc,
	}
}

func (h *RankingHandler) GetArticleRanking(c *gin.Context) {
	rankType := c.DefaultQuery("type", "hot")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	switch rankType {
	case "hot":
		ids, err := h.rankingSvc.GetArticleHotRanking(c.Request.Context(), page, pageSize)
		if err != nil {
			platform.Fail(c, http.StatusInternalServerError, platform.CodeInternalError, err.Error())
			return
		}

		platform.OK(c, gin.H{
			"ids":       ids,
			"page":      page,
			"page_size": pageSize,
		})

	default:
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "不支持的排行类型")
	}
}

func (h *RankingHandler) GetSkillRanking(c *gin.Context) {
	rankType := c.DefaultQuery("type", "hot")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	var ids []uint
	var err error

	switch rankType {
	case "hot":
		ids, err = h.rankingSvc.GetSkillHotRanking(c.Request.Context(), page, pageSize)
	default:
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "不支持的排行类型")
		return
	}

	if err != nil {
		platform.Fail(c, http.StatusInternalServerError, platform.CodeInternalError, err.Error())
		return
	}

	platform.OK(c, gin.H{
		"ids":       ids,
		"page":      page,
		"page_size": pageSize,
	})
}

func (h *RankingHandler) GetMcpServerRanking(c *gin.Context) {
	rankType := c.DefaultQuery("type", "hot")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	var ids []uint
	var err error

	switch rankType {
	case "hot":
		ids, err = h.rankingSvc.GetMcpServerHotRanking(c.Request.Context(), page, pageSize)
	default:
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "不支持的排行类型")
		return
	}

	if err != nil {
		platform.Fail(c, http.StatusInternalServerError, platform.CodeInternalError, err.Error())
		return
	}

	platform.OK(c, gin.H{
		"ids":       ids,
		"page":      page,
		"page_size": pageSize,
	})
}
