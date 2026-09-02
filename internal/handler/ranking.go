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
	rankType := c.DefaultQuery("type", "hot")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	switch rankType {
	case "hot":
		articles, err := h.rankingSvc.ListArticleHot(c.Request.Context(), page, pageSize)
		if err != nil {
			platform.Fail(c, http.StatusInternalServerError, platform.CodeInternalError, err.Error())
			return
		}
		platform.OK(c, gin.H{
			"articles":  articles,
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

	switch rankType {
	case "hot":
		skills, err := h.rankingSvc.ListSkillHot(c.Request.Context(), page, pageSize)
		if err != nil {
			platform.Fail(c, http.StatusInternalServerError, platform.CodeInternalError, err.Error())
			return
		}
		platform.OK(c, gin.H{
			"skills":    skills,
			"page":      page,
			"page_size": pageSize,
		})
	default:
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "不支持的排行类型")
	}
}

func (h *RankingHandler) GetMcpServerRanking(c *gin.Context) {
	rankType := c.DefaultQuery("type", "hot")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	switch rankType {
	case "hot":
		servers, err := h.rankingSvc.ListMcpServerHot(c.Request.Context(), page, pageSize)
		if err != nil {
			platform.Fail(c, http.StatusInternalServerError, platform.CodeInternalError, err.Error())
			return
		}
		platform.OK(c, gin.H{
			"servers":   servers,
			"page":      page,
			"page_size": pageSize,
		})
	default:
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "不支持的排行类型")
	}
}
