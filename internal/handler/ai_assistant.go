package handler

import (
	"aidevclub/internal/platform"
	"aidevclub/internal/service"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

type AIAssistantHandler struct{ svc *service.AIAssistantService }

func NewAIAssistantHandler(svc *service.AIAssistantService) *AIAssistantHandler {
	return &AIAssistantHandler{svc: svc}
}
func (h *AIAssistantHandler) Ask(c *gin.Context) {
	var in struct {
		Question  string `json:"question"`
		ArticleID *uint  `json:"article_id"`
	}
	if c.ShouldBindJSON(&in) != nil || strings.TrimSpace(in.Question) == "" {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "问题不能为空")
		return
	}
	out, err := h.svc.Ask(c.Request.Context(), in.Question, in.ArticleID)
	if err != nil {
		platform.Fail(c, http.StatusBadGateway, platform.CodeInternalError, err.Error())
		return
	}
	platform.OK(c, out)
}
