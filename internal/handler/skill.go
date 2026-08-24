package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"aidevclub/internal/platform"
	"aidevclub/internal/service"
)

type SkillHandler struct{ svc *service.SkillService }

func NewSkillHandler(svc *service.SkillService) *SkillHandler { return &SkillHandler{svc: svc} }

func (h *SkillHandler) Create(c *gin.Context) {
	var in struct {
		Name        string   `json:"name"`
		Description string   `json:"description"`
		RepoURL     string   `json:"repo_url"`
		TagIDs      []uint   `json:"tag_ids"`
		TagNames    []string `json:"tag_names"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "参数错误")
		return
	}
	sk, err := h.svc.Create(c.Request.Context(), c.GetUint("user_id"), service.CreateSkillInput{
		Name: in.Name, Description: in.Description, RepoURL: in.RepoURL,
		TagIDs: in.TagIDs, TagNames: in.TagNames,
	})
	if err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, gin.H{"id": sk.ID})
}

func (h *SkillHandler) Update(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "参数错误")
		return
	}
	var in struct {
		Name        string   `json:"name"`
		Description string   `json:"description"`
		RepoURL     string   `json:"repo_url"`
		TagIDs      []uint   `json:"tag_ids"`
		TagNames    []string `json:"tag_names"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "参数错误")
		return
	}
	sk, err := h.svc.Update(c.Request.Context(), c.GetUint("user_id"), id, service.CreateSkillInput{
		Name: in.Name, Description: in.Description, RepoURL: in.RepoURL,
		TagIDs: in.TagIDs, TagNames: in.TagNames,
	})
	if err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, gin.H{"id": sk.ID})
}

func (h *SkillHandler) Delete(c *gin.Context) {
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

func (h *SkillHandler) Get(c *gin.Context) {
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

func (h *SkillHandler) List(c *gin.Context) {
	q := service.SkillListQuery{
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

func (h *SkillHandler) Submit(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "参数错误")
		return
	}
	sk, err := h.svc.Submit(c.Request.Context(), c.GetUint("user_id"), id)
	if err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, gin.H{"id": sk.ID, "status": string(sk.Status)})
}

func (h *SkillHandler) Withdraw(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "参数错误")
		return
	}
	sk, err := h.svc.Withdraw(c.Request.Context(), c.GetUint("user_id"), id)
	if err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, gin.H{"id": sk.ID, "status": string(sk.Status)})
}

func (h *SkillHandler) Archive(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "参数错误")
		return
	}
	sk, err := h.svc.Archive(c.Request.Context(), c.GetUint("user_id"), id)
	if err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, gin.H{"id": sk.ID, "status": string(sk.Status)})
}
