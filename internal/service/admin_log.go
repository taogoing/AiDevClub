package service

import (
	"context"
	"encoding/json"

	"aidevclub/internal/model"
	"aidevclub/internal/repo"
)

type AdminLogService struct {
	repo  *repo.AdminLogRepo
	users *repo.UserRepo
}

func NewAdminLogService(repo *repo.AdminLogRepo, users *repo.UserRepo) *AdminLogService {
	return &AdminLogService{repo: repo, users: users}
}

func (s *AdminLogService) Log(ctx context.Context, adminID uint, action model.AdminLogAction, targetType string, targetID uint, detail interface{}) error {
	var detailStr string
	if detail != nil {
		b, err := json.Marshal(detail)
		if err != nil {
			return err
		}
		detailStr = string(b)
	}
	log := &model.AdminLog{
		AdminID:    adminID,
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		Detail:     detailStr,
	}
	return s.repo.Create(log)
}

func (s *AdminLogService) List(ctx context.Context, action model.AdminLogAction, page, pageSize int) (*AdminLogListResult, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	list, total, err := s.repo.List(ctx, action, page, pageSize)
	if err != nil {
		return nil, err
	}
	adminIDs := make([]uint, 0, len(list))
	for _, l := range list {
		adminIDs = append(adminIDs, l.AdminID)
	}
	admins, err := s.users.FindPublicByIDs(ctx, adminIDs)
	if err != nil {
		return nil, err
	}
	adminMap := make(map[uint]AuthorBrief)
	for _, a := range admins {
		adminMap[a.ID] = AuthorBrief{ID: a.ID, Nickname: a.Nickname, AvatarURL: a.AvatarURL}
	}
	out := &AdminLogListResult{
		List:  make([]AdminLogItem, 0, len(list)),
		Total: total, Page: page, PageSize: pageSize,
	}
	for _, l := range list {
		var detail any
		if l.Detail != "" {
			if err := json.Unmarshal([]byte(l.Detail), &detail); err != nil {
				detail = l.Detail
			}
		}
		out.List = append(out.List, AdminLogItem{
			ID: l.ID, AdminID: l.AdminID,
			Admin:      adminMap[l.AdminID],
			Action:     string(l.Action),
			TargetType: l.TargetType, TargetID: l.TargetID,
			Detail:    detail,
			CreatedAt: l.CreatedAt,
		})
	}
	return out, nil
}
