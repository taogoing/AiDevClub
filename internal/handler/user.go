package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"aidevclub/internal/platform"
	"aidevclub/internal/service"
)

type UserHandler struct {
	svc *service.UserService
}

func NewUserHandler(svc *service.UserService) *UserHandler { return &UserHandler{svc: svc} }

func (h *UserHandler) Me(c *gin.Context) {
	u, err := h.svc.Get(c.Request.Context(), c.GetUint("user_id"))
	if err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, gin.H{"id": u.ID, "email": u.Email, "nickname": u.Nickname, "avatar_url": u.AvatarURL, "bio": u.Bio})
}

func (h *UserHandler) Update(c *gin.Context) {
	var in struct {
		Nickname  string `json:"nickname"`
		AvatarURL string `json:"avatar_url"`
		Bio       string `json:"bio"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		platform.Fail(c, http.StatusBadRequest, 40001, "参数错误")
		return
	}
	if err := h.svc.UpdateProfile(c.Request.Context(), c.GetUint("user_id"), service.UpdateProfileInput{Nickname: in.Nickname, AvatarURL: in.AvatarURL, Bio: in.Bio}); err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, nil)
}

func (h *UserHandler) ChangePassword(c *gin.Context) {
	var in struct {
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.Password == "" {
		platform.Fail(c, http.StatusBadRequest, 40001, "参数错误")
		return
	}
	if err := h.svc.ChangePassword(c.Request.Context(), c.GetUint("user_id"), in.Password); err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, nil)
}

func (h *UserHandler) Delete(c *gin.Context) {
	if err := h.svc.Delete(c.Request.Context(), c.GetUint("user_id")); err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, nil)
}
