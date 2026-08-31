package handler

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"

	"aidevclub/internal/model"
	"aidevclub/internal/platform"
	"aidevclub/internal/service"
)

type ArticleHandler struct{ svc *service.ArticleService }

func NewArticleHandler(svc *service.ArticleService) *ArticleHandler { return &ArticleHandler{svc: svc} }

func (h *ArticleHandler) Create(c *gin.Context) {
	var in struct {
		Title    string              `json:"title"`
		Summary  string              `json:"summary"`
		Content  string              `json:"content"`
		Status   model.ArticleStatus `json:"status"`
		TagIDs   []uint              `json:"tag_ids"`
		TagNames []string            `json:"tag_names"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "参数错误")
		return
	}
	a, err := h.svc.Create(c.Request.Context(), c.GetUint("user_id"), service.CreateArticleInput{
		Title: in.Title, Summary: in.Summary, Content: in.Content,
		Status: in.Status, TagIDs: in.TagIDs, TagNames: in.TagNames,
	})
	if err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, gin.H{"id": a.ID})
}

func (h *ArticleHandler) Update(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "参数错误")
		return
	}
	var in struct {
		Title    string              `json:"title"`
		Summary  string              `json:"summary"`
		Content  string              `json:"content"`
		Status   model.ArticleStatus `json:"status"`
		TagIDs   []uint              `json:"tag_ids"`
		TagNames []string            `json:"tag_names"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "参数错误")
		return
	}
	a, err := h.svc.Update(c.Request.Context(), c.GetUint("user_id"), id, service.CreateArticleInput{
		Title: in.Title, Summary: in.Summary, Content: in.Content,
		Status: in.Status, TagIDs: in.TagIDs, TagNames: in.TagNames,
	})
	if err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, gin.H{"id": a.ID})
}

func (h *ArticleHandler) Delete(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "参数错误")
		return
	}
	if err := h.svc.Delete(c.Request.Context(), c.GetUint("user_id"), id); err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, nil)
}

func parseUintParam(c *gin.Context, name string) (uint, error) {
	s := c.Param(name)
	var v uint
	_, err := fmt.Sscanf(s, "%d", &v)
	return v, err
}

func queryInt(c *gin.Context, key string, def int) int {
	v := c.Query(key)
	if v == "" {
		return def
	}
	var n int
	fmt.Sscanf(v, "%d", &n)
	if n <= 0 {
		return def
	}
	return n
}

func parseUint(s string) uint {
	var v uint
	fmt.Sscanf(s, "%d", &v)
	return v
}

func (h *ArticleHandler) List(c *gin.Context) {
	q := service.ListQuery{
		Page:     queryInt(c, "page", 1),
		PageSize: queryInt(c, "page_size", 20),
		Keyword:  c.Query("keyword"),
		Sort:     c.Query("sort"),
	}
	if v := c.Query("tag_id"); v != "" {
		id := parseUint(v)
		q.TagID = &id
	}
	if v := c.Query("author_id"); v != "" {
		id := parseUint(v)
		q.AuthorID = &id
	}
	res, err := h.svc.List(c.Request.Context(), q)
	if err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, res)
}

func (h *ArticleHandler) ListMine(c *gin.Context) {
	page := queryInt(c, "page", 1)
	pageSize := queryInt(c, "page_size", 20)
	status := c.Query("status")
	res, err := h.svc.ListOwned(c.Request.Context(), c.GetUint("user_id"), status, page, pageSize)
	if err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, res)
}

func (h *ArticleHandler) Get(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "参数错误")
		return
	}
	detail, err := h.svc.Get(c.Request.Context(), c.GetUint("user_id"), id)
	if err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, detail)
}

func (h *ArticleHandler) Like(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "参数错误")
		return
	}
	liked, count, err := h.svc.ToggleLike(c.Request.Context(), c.GetUint("user_id"), id)
	if err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, gin.H{"liked": liked, "likes_count": count})
}

func (h *ArticleHandler) Favorite(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "参数错误")
		return
	}
	favorited, count, err := h.svc.ToggleFavorite(c.Request.Context(), c.GetUint("user_id"), id)
	if err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, gin.H{"favorited": favorited, "favorites_count": count})
}

func (h *ArticleHandler) UploadImage(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "参数错误")
		return
	}
	ext := strings.ToLower(filepath.Ext(file.Filename))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp", ".gif":
	default:
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "不支持的图片格式")
		return
	}
	if file.Size > h.svc.MaxImageBytes() {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "图片过大")
		return
	}
	if err := os.MkdirAll(h.svc.ImageDir(), 0o755); err != nil {
		platform.Fail(c, http.StatusInternalServerError, platform.CodeInternalError, "服务器内部错误")
		return
	}
	name := randomHex(16) + ext
	if err := c.SaveUploadedFile(file, filepath.Join(h.svc.ImageDir(), name)); err != nil {
		platform.Fail(c, http.StatusInternalServerError, platform.CodeInternalError, "服务器内部错误")
		return
	}
	platform.OK(c, gin.H{"url": "/static/articles/" + name})
}
