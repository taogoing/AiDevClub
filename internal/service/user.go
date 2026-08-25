package service

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"aidevclub/internal/model"
	"aidevclub/internal/platform"
	"aidevclub/internal/repo"
)

type UpdateProfileInput struct {
	Nickname  string
	AvatarURL string
	Bio       string
}

type UserService struct {
	users  *repo.UserRepo
	tokens *repo.TokenRepo
	cfg    *platform.Config
}

func NewUserService(users *repo.UserRepo, tokens *repo.TokenRepo, cfg *platform.Config) *UserService {
	return &UserService{users: users, tokens: tokens, cfg: cfg}
}

func (s *UserService) AvatarDir() string     { return s.cfg.AvatarDir }
func (s *UserService) MaxAvatarBytes() int64 { return s.cfg.MaxAvatarBytes }

func (s *UserService) Get(ctx context.Context, id uint) (*model.User, error) {
	u, err := s.users.FindByIDWithContext(ctx, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrUserNotFound
	}
	return u, err
}

func (s *UserService) UpdateProfile(ctx context.Context, id uint, in UpdateProfileInput) error {
	u, err := s.users.FindByID(id)
	if err != nil {
		return ErrUserNotFound
	}
	if in.Nickname != "" {
		u.Nickname = in.Nickname
	}
	if in.AvatarURL != "" {
		u.AvatarURL = in.AvatarURL
	}
	if in.Bio != "" {
		u.Bio = in.Bio
	}
	return s.users.Update(u)
}

func (s *UserService) ChangePassword(ctx context.Context, id uint, newPassword string) error {
	u, err := s.users.FindByID(id)
	if err != nil {
		return ErrUserNotFound
	}
	hash, err := hashPassword(newPassword)
	if err != nil {
		return err
	}
	u.PasswordHash = hash
	if err := s.users.Update(u); err != nil {
		return err
	}
	return s.tokens.RevokeAllForUser(ctx, id)
}

func (s *UserService) Delete(ctx context.Context, id uint) error {
	if err := s.users.Delete(id); err != nil {
		return err
	}
	return s.tokens.RevokeAllForUser(ctx, id)
}
