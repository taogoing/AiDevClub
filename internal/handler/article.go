package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"aidevclub/internal/model"
	"aidevclub/internal/platform"
	"aidevclub/internal/service"
)

type ArticleHandler struct{ svc *service.ArticleService }

func NewArticleHandler(svc *service.ArticleService) *ArticleHandler { return &ArticleHandler{svc: svc} }

func (h *ArticleHandler) Create(c *gin.Context) {
	var in struct {
		Title      string              `json:"title"`
		Summary    string              `json:"summary"`
		Content    string              `json:"content"`
		CategoryID uint                `json:"category_id"`
		Status     model.ArticleStatus `json:"status"`
		TagIDs     []uint              `json:"tag_ids"`
		TagNames   []string            `json:"tag_names"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		platform.Fail(c, http.StatusBadRequest, 40001, "参数错误")
		return
	}
	a, err := h.svc.Create(c.Request.Context(), c.GetUint("user_id"), service.CreateArticleInput{
		Title: in.Title, Summary: in.Summary, Content: in.Content,
		CategoryID: in.CategoryID, Status: in.Status, TagIDs: in.TagIDs, TagNames: in.TagNames,
	})
	if err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, gin.H{"id": a.ID})
}
