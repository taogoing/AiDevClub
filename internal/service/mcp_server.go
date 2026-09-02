package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"gorm.io/gorm"

	"aidevclub/internal/model"
	"aidevclub/internal/platform"
	"aidevclub/internal/repo"
)

var (
	ErrMcpServerNotFound = platform.NewBizError(http.StatusNotFound, platform.CodeMcpServerNotFound, "MCP Server 不存在或不可见")
	ErrMcpServerState    = platform.NewBizError(http.StatusBadRequest, platform.CodeStateError, "当前状态不允许此操作")
)

type McpServerService struct {
	servers  *repo.McpServerRepo
	tags     *repo.TagRepo
	inter    *repo.InteractionRepo
	cfg      *platform.Config
	notifSvc *NotificationService
}

func NewMcpServerService(servers *repo.McpServerRepo, tags *repo.TagRepo, inter *repo.InteractionRepo, cfg *platform.Config, notifSvc *NotificationService) *McpServerService {
	return &McpServerService{servers: servers, tags: tags, inter: inter, cfg: cfg, notifSvc: notifSvc}
}

var supportedMcpClients = map[string]bool{
	"cursor": true, "claude-code": true, "codex": true,
	"trae": true, "trae-global": true, "cline": true, "windsurf": true,
}

func normalizeMcpInstallations(installations []McpInstallation) (string, error) {
	if len(installations) > 12 {
		return "", ErrBadParam
	}
	seen := make(map[string]bool, len(installations))
	for i := range installations {
		item := &installations[i]
		item.Client = strings.TrimSpace(item.Client)
		item.Command = strings.TrimSpace(item.Command)
		if !supportedMcpClients[item.Client] || seen[item.Client] || len(item.Command) > 4000 {
			return "", ErrBadParam
		}
		if item.Command == "" && len(item.Config) == 0 && len(item.WindowsConfig) == 0 {
			return "", ErrBadParam
		}
		seen[item.Client] = true
	}
	b, err := json.Marshal(installations)
	if err != nil || len(b) > 128*1024 {
		return "", ErrBadParam
	}
	return string(b), nil
}

func (s *McpServerService) ResolveTagSet(ctx context.Context, tx *gorm.DB, tagIDs []uint, tagNames []string) ([]uint, error) {
	set := map[uint]bool{}
	var out []uint
	for _, id := range tagIDs {
		t, err := s.tags.FindByID(ctx, id)
		if err != nil || !t.Enabled {
			return nil, ErrTagNotFound
		}
		if !set[id] {
			set[id] = true
			out = append(out, id)
		}
	}
	for _, name := range tagNames {
		if name == "" {
			continue
		}
		t, err := s.tags.FindByName(ctx, name)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			t, err = s.tags.Create(ctx, name)
			if err != nil {
				if platform.IsDuplicateEntry(err) {
					t, err = s.tags.FindByName(ctx, name)
				} else {
					return nil, err
				}
			}
		} else if err != nil {
			return nil, err
		}
		if !t.Enabled {
			continue
		}
		if !set[t.ID] {
			set[t.ID] = true
			out = append(out, t.ID)
		}
	}
	return out, nil
}

func (s *McpServerService) Create(ctx context.Context, userID uint, in CreateMcpServerInput) (*model.McpServer, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Description = strings.TrimSpace(in.Description)
	in.RepoURL = strings.TrimSpace(in.RepoURL)
	if in.Name == "" {
		return nil, ErrBadParam
	}
	if len([]rune(in.Name)) > 100 || len([]rune(in.Description)) > 500 || (in.RepoURL != "" && !validRepositoryURL(in.RepoURL)) {
		return nil, ErrBadParam
	}
	installationsJSON, err := normalizeMcpInstallations(in.Installations)
	if err != nil {
		return nil, err
	}
	sv := &model.McpServer{
		AuthorID:          userID,
		Name:              in.Name,
		Description:       in.Description,
		RepoURL:           in.RepoURL,
		InstallationsJSON: installationsJSON,
		Readme:            in.Readme,
		Status:            model.ResourceStatusDraft,
	}
	err = s.servers.DB().Transaction(func(tx *gorm.DB) error {
		if err := s.servers.Create(tx, sv); err != nil {
			return err
		}
		tagIDs, err := s.ResolveTagSet(ctx, tx, in.TagIDs, in.TagNames)
		if err != nil {
			return err
		}
		if len(tagIDs) > 0 {
			if err := s.servers.SetMcpServerTags(tx, sv.ID, tagIDs); err != nil {
				return err
			}
			for _, id := range tagIDs {
				if err := s.tags.IncrUsage(tx, id, 1); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return sv, nil
}

func (s *McpServerService) Update(ctx context.Context, userID, serverID uint, in CreateMcpServerInput) (*model.McpServer, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Description = strings.TrimSpace(in.Description)
	in.RepoURL = strings.TrimSpace(in.RepoURL)
	if in.Name == "" {
		return nil, ErrBadParam
	}
	if len([]rune(in.Name)) > 100 || len([]rune(in.Description)) > 500 || (in.RepoURL != "" && !validRepositoryURL(in.RepoURL)) {
		return nil, ErrBadParam
	}
	installationsJSON, err := normalizeMcpInstallations(in.Installations)
	if err != nil {
		return nil, err
	}
	sv, err := s.servers.FindByID(nil, serverID)
	if err != nil {
		return nil, ErrMcpServerNotFound
	}
	if sv.AuthorID != userID {
		return nil, ErrForbidden
	}
	if sv.Status == model.ResourceStatusPendingReview {
		return nil, ErrMcpServerState
	}
	oldTags, err := s.servers.FindMcpServerTags(nil, serverID)
	if err != nil {
		return nil, err
	}

	sv.Name = in.Name
	sv.Description = in.Description
	sv.RepoURL = in.RepoURL
	sv.InstallationsJSON = installationsJSON
	sv.Readme = in.Readme

	err = s.servers.DB().Transaction(func(tx *gorm.DB) error {
		if err := s.servers.Update(tx, sv); err != nil {
			return err
		}
		newTags, err := s.ResolveTagSet(ctx, tx, in.TagIDs, in.TagNames)
		if err != nil {
			return err
		}
		oldSet := map[uint]bool{}
		for _, id := range oldTags {
			oldSet[id] = true
		}
		newSet := map[uint]bool{}
		for _, id := range newTags {
			newSet[id] = true
		}
		for _, id := range oldTags {
			if !newSet[id] {
				if err := s.tags.IncrUsage(tx, id, -1); err != nil {
					return err
				}
			}
		}
		for _, id := range newTags {
			if !oldSet[id] {
				if err := s.tags.IncrUsage(tx, id, 1); err != nil {
					return err
				}
			}
		}
		return s.servers.SetMcpServerTags(tx, serverID, newTags)
	})
	if err != nil {
		return nil, err
	}
	return sv, nil
}

func (s *McpServerService) Delete(ctx context.Context, userID, serverID uint) error {
	sv, err := s.servers.FindByID(nil, serverID)
	if err != nil {
		return ErrMcpServerNotFound
	}
	if sv.AuthorID != userID {
		return ErrForbidden
	}
	return s.servers.DB().Transaction(func(tx *gorm.DB) error {
		tagIDs, err := s.servers.FindMcpServerTags(tx, serverID)
		if err != nil {
			return err
		}
		if err := s.servers.Delete(tx, serverID); err != nil {
			return err
		}
		for _, id := range tagIDs {
			if err := s.tags.IncrUsage(tx, id, -1); err != nil {
				return err
			}
		}
		return s.servers.SetMcpServerTags(tx, serverID, nil)
	})
}

func (s *McpServerService) Submit(ctx context.Context, userID, serverID uint) (*model.McpServer, error) {
	sv, err := s.servers.FindByID(nil, serverID)
	if err != nil {
		return nil, ErrMcpServerNotFound
	}
	if sv.AuthorID != userID {
		return nil, ErrForbidden
	}
	var installations []McpInstallation
	if !validRepositoryURL(sv.RepoURL) || json.Unmarshal([]byte(sv.InstallationsJSON), &installations) != nil || len(installations) == 0 {
		return nil, ErrBadParam
	}
	switch sv.Status {
	case model.ResourceStatusDraft, model.ResourceStatusRejected, model.ResourceStatusArchived:
	default:
		return nil, ErrMcpServerState
	}
	sv.Status = model.ResourceStatusPendingReview
	if err := s.servers.Update(nil, sv); err != nil {
		return nil, err
	}
	return sv, nil
}

func (s *McpServerService) Withdraw(ctx context.Context, userID, serverID uint) (*model.McpServer, error) {
	sv, err := s.servers.FindByID(nil, serverID)
	if err != nil {
		return nil, ErrMcpServerNotFound
	}
	if sv.AuthorID != userID {
		return nil, ErrForbidden
	}
	if sv.Status != model.ResourceStatusPendingReview {
		return nil, ErrMcpServerState
	}
	sv.Status = model.ResourceStatusDraft
	if err := s.servers.Update(nil, sv); err != nil {
		return nil, err
	}
	return sv, nil
}

func (s *McpServerService) Archive(ctx context.Context, userID, serverID uint) (*model.McpServer, error) {
	sv, err := s.servers.FindByID(nil, serverID)
	if err != nil {
		return nil, ErrMcpServerNotFound
	}
	if sv.AuthorID != userID {
		return nil, ErrForbidden
	}
	if sv.Status != model.ResourceStatusPublished {
		return nil, ErrMcpServerState
	}
	sv.Status = model.ResourceStatusArchived
	if err := s.servers.Update(nil, sv); err != nil {
		return nil, err
	}
	return sv, nil
}

func (s *McpServerService) canView(sv *model.McpServer, userID uint) bool {
	if userID > 0 && sv.AuthorID == userID {
		return true
	}
	return sv.Status == model.ResourceStatusPublished && !sv.Hidden
}

func (s *McpServerService) summaryOf(sv model.McpServer, tags []model.Tag) McpServerSummary {
	sm := McpServerSummary{
		ID: sv.ID, Name: sv.Name, Description: sv.Description,
		RepoURL: sv.RepoURL, Tags: []TagBrief{},
		Views:      sv.Views,
		LikesCount: sv.LikesCount, FavoritesCount: sv.FavoritesCount,
		CommentsCount: sv.CommentsCount, Status: string(sv.Status),
		PublishedAt: sv.PublishedAt,
		Author:      AuthorBrief{ID: sv.AuthorID},
	}
	if sv.Author != nil {
		sm.Author = AuthorBrief{ID: sv.Author.ID, Nickname: sv.Author.Nickname, AvatarURL: sv.Author.AvatarURL}
	}
	for _, t := range tags {
		sm.Tags = append(sm.Tags, TagBrief{ID: t.ID, Name: t.Name})
	}
	return sm
}

func (s *McpServerService) Get(ctx context.Context, userID, serverID uint) (*McpServerDetail, error) {
	return s.detail(ctx, userID, serverID, true, true)
}

func (s *McpServerService) Read(ctx context.Context, userID, serverID uint) (*McpServerDetail, error) {
	return s.detail(ctx, userID, serverID, false, false)
}

func (s *McpServerService) detail(ctx context.Context, userID, serverID uint, trackView, loadInteractions bool) (*McpServerDetail, error) {
	sv, err := s.servers.FindByIDWithContext(ctx, serverID)
	if err != nil {
		return nil, ErrMcpServerNotFound
	}
	if !s.canView(sv, userID) {
		return nil, ErrMcpServerNotFound
	}
	if trackView && sv.Status == model.ResourceStatusPublished {
		_ = s.servers.IncrViews(ctx, serverID)
		sv.Views++
	}
	tagMap, err := s.servers.TagsForMcpServers(ctx, []uint{serverID})
	if err != nil {
		return nil, err
	}
	sm := s.summaryOf(*sv, tagMap[serverID])
	d := &McpServerDetail{
		McpServerSummary: sm,
		Installations:    []McpInstallation{},
		Readme:           sv.Readme,
	}
	if strings.TrimSpace(sv.InstallationsJSON) != "" {
		if err := json.Unmarshal([]byte(sv.InstallationsJSON), &d.Installations); err != nil {
			return nil, err
		}
	}
	if loadInteractions && userID > 0 {
		if d.Liked, err = s.inter.McpServerLiked(ctx, userID, serverID); err != nil {
			return nil, err
		}
		if d.Favorited, err = s.inter.McpServerFavorited(ctx, userID, serverID); err != nil {
			return nil, err
		}
	}
	return d, nil
}

func (s *McpServerService) ToggleLike(ctx context.Context, userID, serverID uint) (bool, int, error) {
	sv, err := s.servers.FindByID(nil, serverID)
	if err != nil || sv.Status != model.ResourceStatusPublished {
		return false, 0, ErrMcpServerNotFound
	}
	var liked bool
	var newCount int
	err = s.servers.DB().Transaction(func(tx *gorm.DB) error {
		var err error
		liked, err = s.inter.ToggleMcpServerLike(tx, userID, serverID)
		if err != nil {
			return err
		}
		delta := 1
		if !liked {
			delta = -1
		}
		if err := s.servers.IncrCount(tx, serverID, "likes_count", delta); err != nil {
			return err
		}
		newCount = sv.LikesCount + delta
		return nil
	})
	if err == nil {
		if liked {
			go func() {
				_ = s.notifSvc.Create(context.Background(), sv.AuthorID, model.NotifTypeLikeMcpServer, "点赞", "有人赞了你的 MCP Server", "mcp_server", serverID, userID)
			}()
		}
	}
	return liked, newCount, err
}

func (s *McpServerService) ToggleFavorite(ctx context.Context, userID, serverID uint) (bool, int, error) {
	sv, err := s.servers.FindByID(nil, serverID)
	if err != nil || sv.Status != model.ResourceStatusPublished {
		return false, 0, ErrMcpServerNotFound
	}
	var favorited bool
	var newCount int
	err = s.servers.DB().Transaction(func(tx *gorm.DB) error {
		var err error
		favorited, err = s.inter.ToggleMcpServerFavorite(tx, userID, serverID)
		if err != nil {
			return err
		}
		delta := 1
		if !favorited {
			delta = -1
		}
		if err := s.servers.IncrCount(tx, serverID, "favorites_count", delta); err != nil {
			return err
		}
		newCount = sv.FavoritesCount + delta
		return nil
	})
	return favorited, newCount, err
}

func (s *McpServerService) List(ctx context.Context, q McpServerListQuery) (*McpServerListResult, error) {
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 {
		q.PageSize = s.cfg.DefaultPageSize
	}
	if q.PageSize > s.cfg.MaxPageSize {
		q.PageSize = s.cfg.MaxPageSize
	}
	switch q.Sort {
	case "hot":
	default:
		q.Sort = "latest"
	}
	rq := repo.McpServerQuery{
		Page: q.Page, PageSize: q.PageSize,
		TagID: q.TagID, Keyword: q.Keyword, AuthorID: q.AuthorID, Sort: q.Sort,
	}

	list, total, err := s.servers.List(ctx, rq)
	if err != nil {
		return nil, err
	}
	ids := make([]uint, 0, len(list))
	for _, sv := range list {
		ids = append(ids, sv.ID)
	}
	tagMap, err := s.servers.TagsForMcpServers(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := &McpServerListResult{
		List:  make([]McpServerSummary, 0, len(list)),
		Total: total, Page: q.Page, PageSize: q.PageSize,
	}
	for _, sv := range list {
		out.List = append(out.List, s.summaryOf(sv, tagMap[sv.ID]))
	}
	return out, nil
}

func (s *McpServerService) ListOwned(ctx context.Context, userID uint, status string, page, pageSize int) (*McpServerListResult, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = s.cfg.DefaultPageSize
	}
	if pageSize > s.cfg.MaxPageSize {
		pageSize = s.cfg.MaxPageSize
	}
	list, total, err := s.servers.ListOwned(ctx, userID, status, page, pageSize)
	if err != nil {
		return nil, err
	}
	ids := make([]uint, 0, len(list))
	for _, server := range list {
		ids = append(ids, server.ID)
	}
	tagMap, err := s.servers.TagsForMcpServers(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := &McpServerListResult{List: make([]McpServerSummary, 0, len(list)), Total: total, Page: page, PageSize: pageSize}
	for _, server := range list {
		out.List = append(out.List, s.summaryOf(server, tagMap[server.ID]))
	}
	return out, nil
}
