package handler

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"os"
	"path/filepath"
	"strings"

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
	platform.OK(c, gin.H{"id": u.ID, "email": u.Email, "nickname": u.Nickname, "avatar_url": u.AvatarURL, "bio": u.Bio, "role": u.Role})
}

func (h *UserHandler) Update(c *gin.Context) {
	var in struct {
		Nickname  string `json:"nickname"`
		AvatarURL string `json:"avatar_url"`
		Bio       string `json:"bio"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "参数错误")
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
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "参数错误")
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

func (h *UserHandler) UploadAvatar(c *gin.Context) {
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
	if file.Size > h.svc.MaxAvatarBytes() {
		platform.Fail(c, http.StatusBadRequest, platform.CodeParamError, "图片过大")
		return
	}
	if err := os.MkdirAll(h.svc.AvatarDir(), 0o755); err != nil {
		platform.Fail(c, http.StatusInternalServerError, platform.CodeInternalError, "服务器内部错误")
		return
	}
	name := randomHex(16) + ext
	if err := c.SaveUploadedFile(file, filepath.Join(h.svc.AvatarDir(), name)); err != nil {
		platform.Fail(c, http.StatusInternalServerError, platform.CodeInternalError, "服务器内部错误")
		return
	}
	url := "/static/avatars/" + name
	if err := h.svc.UpdateProfile(c.Request.Context(), c.GetUint("user_id"), service.UpdateProfileInput{AvatarURL: url}); err != nil {
		platform.Fail(c, errStatus(err), errCode(err), err.Error())
		return
	}
	platform.OK(c, gin.H{"avatar_url": url})
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
