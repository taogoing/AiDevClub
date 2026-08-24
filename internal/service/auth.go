package service

import (
	"context"
	"errors"
	"math/rand"
	"net/http"

	"golang.org/x/crypto/bcrypt"

	"aidevclub/internal/model"
	"aidevclub/internal/platform"
	"aidevclub/internal/repo"
)

func hashPassword(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	return string(b), err
}

func checkPassword(hash, plain string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
}

var (
	ErrEmailExists   = platform.NewBizError(http.StatusConflict, 40901, "邮箱已存在")
	ErrUserNotFound  = platform.NewBizError(http.StatusNotFound, 40401, "用户不存在")
	ErrBadCredential = platform.NewBizError(http.StatusBadRequest, 40001, "邮箱或密码错误")
	ErrInvalidParam  = platform.NewBizError(http.StatusBadRequest, 40001, "参数错误")
)

type RegisterInput struct {
	Email    string
	Password string
	Nickname string
}

type LoginInput struct {
	Email    string
	Password string
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

type AuthService struct {
	users  *repo.UserRepo
	tokens *repo.TokenRepo
	cfg    *platform.Config
}

func NewAuthService(users *repo.UserRepo, tokens *repo.TokenRepo, cfg *platform.Config) *AuthService {
	return &AuthService{users: users, tokens: tokens, cfg: cfg}
}

func defaultNickname() string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 6)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return "用户_" + string(b)
}

func (s *AuthService) Register(ctx context.Context, in RegisterInput) error {
	if in.Email == "" || in.Password == "" {
		return ErrInvalidParam
	}
	if _, err := s.users.FindByEmail(in.Email); err == nil {
		return ErrEmailExists
	}
	hash, err := hashPassword(in.Password)
	if err != nil {
		return err
	}
	nickname := in.Nickname
	if nickname == "" {
		nickname = defaultNickname()
	}
	if err := s.users.Create(&model.User{
		Email:        in.Email,
		PasswordHash: hash,
		Nickname:     nickname,
		AvatarURL:    s.cfg.DefaultAvatarURL,
	}); err != nil {
		if platform.IsDuplicateEntry(err) {
			return ErrEmailExists
		}
		return err
	}
	return nil
}

func (s *AuthService) Login(ctx context.Context, in LoginInput) (*TokenPair, error) {
	if in.Email == "" || in.Password == "" {
		return nil, ErrInvalidParam
	}
	u, err := s.users.FindByEmail(in.Email)
	if err != nil {
		return nil, ErrBadCredential
	}
	if err := checkPassword(u.PasswordHash, in.Password); err != nil {
		return nil, ErrBadCredential
	}
	return s.issueTokens(ctx, u.ID)
}

func (s *AuthService) issueTokens(ctx context.Context, userID uint) (*TokenPair, error) {
	access, err := platform.GenerateAccessToken(s.cfg.JWTSecret, s.cfg.AccessTokenTTL, userID)
	if err != nil {
		return nil, err
	}
	refresh, err := s.tokens.Issue(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &TokenPair{AccessToken: access, RefreshToken: refresh}, nil
}

func (s *AuthService) Refresh(ctx context.Context, refresh string) (*TokenPair, error) {
	userID, err := s.tokens.Validate(ctx, refresh)
	if err != nil {
		return nil, ErrInvalidParam
	}
	if err := s.tokens.Revoke(ctx, refresh); err != nil {
		return nil, err
	}
	return s.issueTokens(ctx, userID)
}

func (s *AuthService) Logout(ctx context.Context, refresh string) error {
	if err := s.tokens.Revoke(ctx, refresh); err != nil && !errors.Is(err, repo.ErrTokenNotFound) {
		return err
	}
	// 已吊销或已过期的 token 视为登出成功（幂等），避免重复登出被映射成 500。
	return nil
}
