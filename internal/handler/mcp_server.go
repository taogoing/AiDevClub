package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"aidevclub/internal/platform"
	"aidevclub/internal/service"
)

type McpServerHandler struct{ svc *service.McpServerService }

func NewMcpServerHandler(svc *service.McpServerService) *McpServerHandler {
	return &McpServerHandler{svc: svc}
}

func (h *McpServerHandler) Create(c *gin.Context) {
	var in struct {
		Name          string                    `json:"name"`
		Description   string                    `json:"description"`
		RepoURL       string                    `json:"repo_url"`
		Installations []service.McpInstallation `json:"installations"`
		Readme        string                    `json:"readme"`
		TagIDs        []uint                    `json:"tag_ids"`
		TagNames      []string                  `json:"tag_names"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "参数错误")
		return
	}
	sv, err := h.svc.Create(c.Request.Context(), c.GetUint("user_id"), service.CreateMcpServerInput{
		Name: in.Name, Description: in.Description, RepoURL: in.RepoURL,
		Installations: in.Installations, Readme: in.Readme,
		TagIDs: in.TagIDs, TagNames: in.TagNames,
	})
	if err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, gin.H{"id": sv.ID})
}

func (h *McpServerHandler) Update(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "参数错误")
		return
	}
	var in struct {
		Name          string                    `json:"name"`
		Description   string                    `json:"description"`
		RepoURL       string                    `json:"repo_url"`
		Installations []service.McpInstallation `json:"installations"`
		Readme        string                    `json:"readme"`
		TagIDs        []uint                    `json:"tag_ids"`
		TagNames      []string                  `json:"tag_names"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "参数错误")
		return
	}
	sv, err := h.svc.Update(c.Request.Context(), c.GetUint("user_id"), id, service.CreateMcpServerInput{
		Name: in.Name, Description: in.Description, RepoURL: in.RepoURL,
		Installations: in.Installations, Readme: in.Readme,
		TagIDs: in.TagIDs, TagNames: in.TagNames,
	})
	if err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, gin.H{"id": sv.ID})
}

func (h *McpServerHandler) Delete(c *gin.Context) {
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

func (h *McpServerHandler) Get(c *gin.Context) {
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

func (h *McpServerHandler) List(c *gin.Context) {
	q := service.McpServerListQuery{
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

func (h *McpServerHandler) Submit(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "参数错误")
		return
	}
	sv, err := h.svc.Submit(c.Request.Context(), c.GetUint("user_id"), id)
	if err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, gin.H{"id": sv.ID, "status": string(sv.Status)})
}

func (h *McpServerHandler) Withdraw(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "参数错误")
		return
	}
	sv, err := h.svc.Withdraw(c.Request.Context(), c.GetUint("user_id"), id)
	if err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, gin.H{"id": sv.ID, "status": string(sv.Status)})
}

func (h *McpServerHandler) Archive(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "参数错误")
		return
	}
	sv, err := h.svc.Archive(c.Request.Context(), c.GetUint("user_id"), id)
	if err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, gin.H{"id": sv.ID, "status": string(sv.Status)})
}

func (h *McpServerHandler) Like(c *gin.Context) {
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

func (h *McpServerHandler) Favorite(c *gin.Context) {
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
