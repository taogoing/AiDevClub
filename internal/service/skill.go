package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"aidevclub/internal/model"
	"aidevclub/internal/platform"
	"aidevclub/internal/repo"
)

var (
	ErrSkillNotFound = platform.NewBizError(http.StatusNotFound, platform.CodeSkillNotFound, "Skill 不存在或不可见")
	ErrSkillState    = platform.NewBizError(http.StatusBadRequest, platform.CodeStateError, "当前状态不允许此操作")
)

type SkillService struct {
	skills   *repo.SkillRepo
	tags     *repo.TagRepo
	inter    *repo.InteractionRepo
	rdb      *redis.Client
	cfg      *platform.Config
	notifSvc *NotificationService
}

func NewSkillService(skills *repo.SkillRepo, tags *repo.TagRepo, inter *repo.InteractionRepo, rdb *redis.Client, cfg *platform.Config, notifSvc *NotificationService) *SkillService {
	return &SkillService{skills: skills, tags: tags, inter: inter, rdb: rdb, cfg: cfg, notifSvc: notifSvc}
}

func (s *SkillService) ZipDir() string     { return s.cfg.SkillZipDir }
func (s *SkillService) MaxZipBytes() int64 { return s.cfg.MaxResourceZipBytes }

func (s *SkillService) ResolveTagSet(ctx context.Context, tx *gorm.DB, tagIDs []uint, tagNames []string) ([]uint, error) {
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

func (s *SkillService) Create(ctx context.Context, userID uint, in CreateSkillInput) (*model.Skill, error) {
	if in.Name == "" {
		return nil, ErrBadParam
	}
	if len(in.Name) > 100 {
		return nil, ErrBadParam
	}
	sk := &model.Skill{
		AuthorID:    userID,
		Name:        in.Name,
		Description: in.Description,
		RepoURL:     in.RepoURL,
		Status:      model.ResourceStatusDraft,
	}
	err := s.skills.DB().Transaction(func(tx *gorm.DB) error {
		if err := s.skills.Create(tx, sk); err != nil {
			return err
		}
		tagIDs, err := s.ResolveTagSet(ctx, tx, in.TagIDs, in.TagNames)
		if err != nil {
			return err
		}
		if len(tagIDs) > 0 {
			if err := s.skills.SetSkillTags(tx, sk.ID, tagIDs); err != nil {
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
	return sk, nil
}

func (s *SkillService) Update(ctx context.Context, userID, skillID uint, in CreateSkillInput) (*model.Skill, error) {
	if in.Name == "" {
		return nil, ErrBadParam
	}
	if len(in.Name) > 100 {
		return nil, ErrBadParam
	}
	sk, err := s.skills.FindByID(nil, skillID)
	if err != nil {
		return nil, ErrSkillNotFound
	}
	if sk.AuthorID != userID {
		return nil, ErrForbidden
	}
	if sk.Status == model.ResourceStatusPendingReview {
		return nil, ErrSkillState
	}
	oldTags, err := s.skills.FindSkillTags(nil, skillID)
	if err != nil {
		return nil, err
	}

	sk.Name = in.Name
	sk.Description = in.Description
	sk.RepoURL = in.RepoURL

	err = s.skills.DB().Transaction(func(tx *gorm.DB) error {
		if err := s.skills.Update(tx, sk); err != nil {
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
		return s.skills.SetSkillTags(tx, skillID, newTags)
	})
	if err != nil {
		return nil, err
	}
	return sk, nil
}

func (s *SkillService) Delete(ctx context.Context, userID, skillID uint) error {
	sk, err := s.skills.FindByID(nil, skillID)
	if err != nil {
		return ErrSkillNotFound
	}
	if sk.AuthorID != userID {
		return ErrForbidden
	}
	return s.skills.DB().Transaction(func(tx *gorm.DB) error {
		tagIDs, err := s.skills.FindSkillTags(tx, skillID)
		if err != nil {
			return err
		}
		if err := s.skills.Delete(tx, skillID); err != nil {
			return err
		}
		for _, id := range tagIDs {
			if err := s.tags.IncrUsage(tx, id, -1); err != nil {
				return err
			}
		}
		return s.skills.SetSkillTags(tx, skillID, nil)
	})
}

func (s *SkillService) Submit(ctx context.Context, userID, skillID uint) (*model.Skill, error) {
	sk, err := s.skills.FindByID(nil, skillID)
	if err != nil {
		return nil, ErrSkillNotFound
	}
	if sk.AuthorID != userID {
		return nil, ErrForbidden
	}
	switch sk.Status {
	case model.ResourceStatusDraft, model.ResourceStatusRejected, model.ResourceStatusArchived:
	default:
		return nil, ErrSkillState
	}
	sk.Status = model.ResourceStatusPendingReview
	if err := s.skills.Update(nil, sk); err != nil {
		return nil, err
	}
	return sk, nil
}

func (s *SkillService) Withdraw(ctx context.Context, userID, skillID uint) (*model.Skill, error) {
	sk, err := s.skills.FindByID(nil, skillID)
	if err != nil {
		return nil, ErrSkillNotFound
	}
	if sk.AuthorID != userID {
		return nil, ErrForbidden
	}
	if sk.Status != model.ResourceStatusPendingReview {
		return nil, ErrSkillState
	}
	sk.Status = model.ResourceStatusDraft
	if err := s.skills.Update(nil, sk); err != nil {
		return nil, err
	}
	return sk, nil
}

func (s *SkillService) Archive(ctx context.Context, userID, skillID uint) (*model.Skill, error) {
	sk, err := s.skills.FindByID(nil, skillID)
	if err != nil {
		return nil, ErrSkillNotFound
	}
	if sk.AuthorID != userID {
		return nil, ErrForbidden
	}
	if sk.Status != model.ResourceStatusPublished {
		return nil, ErrSkillState
	}
	sk.Status = model.ResourceStatusArchived
	if err := s.skills.Update(nil, sk); err != nil {
		return nil, err
	}
	return sk, nil
}

func (s *SkillService) canView(sk *model.Skill, userID uint) bool {
	if sk.Status == model.ResourceStatusPublished {
		return true
	}
	return sk.AuthorID == userID
}

func (s *SkillService) summaryOf(sk model.Skill, tags []model.Tag) SkillSummary {
	sm := SkillSummary{
		ID: sk.ID, Name: sk.Name, Description: sk.Description,
		RepoURL: sk.RepoURL, Tags: []TagBrief{},
		Views: sk.Views, Downloads: sk.Downloads,
		LikesCount: sk.LikesCount, FavoritesCount: sk.FavoritesCount,
		CommentsCount: sk.CommentsCount, Status: string(sk.Status),
		PublishedAt: sk.PublishedAt,
		Author:      AuthorBrief{ID: sk.AuthorID},
	}
	if sk.Author != nil {
		sm.Author = AuthorBrief{ID: sk.Author.ID, Nickname: sk.Author.Nickname, AvatarURL: sk.Author.AvatarURL}
	}
	for _, t := range tags {
		sm.Tags = append(sm.Tags, TagBrief{ID: t.ID, Name: t.Name})
	}
	return sm
}

func (s *SkillService) Get(ctx context.Context, userID, skillID uint) (*SkillDetail, error) {
	return s.detail(ctx, userID, skillID, true, true, false)
}

func (s *SkillService) Read(ctx context.Context, userID, skillID uint) (*SkillDetail, error) {
	return s.detail(ctx, userID, skillID, false, false, true)
}

func (s *SkillService) detail(ctx context.Context, userID, skillID uint, trackView, loadInteractions, strictVisibility bool) (*SkillDetail, error) {
	sk, err := s.skills.FindByIDWithContext(ctx, skillID)
	if err != nil {
		return nil, ErrSkillNotFound
	}
	if strictVisibility {
		if sk.AuthorID != userID && (sk.Status != model.ResourceStatusPublished || sk.Hidden) {
			return nil, ErrSkillNotFound
		}
	} else if !s.canView(sk, userID) {
		return nil, ErrSkillNotFound
	}
	if trackView && sk.Status == model.ResourceStatusPublished {
		_ = s.skills.IncrViews(ctx, skillID)
		sk.Views++
	}
	tagMap, err := s.skills.TagsForSkills(ctx, []uint{skillID})
	if err != nil {
		return nil, err
	}
	sm := s.summaryOf(*sk, tagMap[skillID])
	d := &SkillDetail{
		SkillSummary: sm,
		ZipURL:       sk.ZipURL,
		ZipFilename:  sk.ZipFilename,
		FileSize:     sk.FileSize,
	}
	if loadInteractions && userID > 0 {
		if d.Liked, err = s.inter.SkillLiked(ctx, userID, skillID); err != nil {
			return nil, err
		}
		if d.Favorited, err = s.inter.SkillFavorited(ctx, userID, skillID); err != nil {
			return nil, err
		}
	}
	return d, nil
}

func (s *SkillService) UploadZip(ctx context.Context, userID, skillID uint, zipURL, zipFilename string, fileSize int64) error {
	zipPath := filepath.Join(s.ZipDir(), filepath.Base(zipURL))
	originalZipPath := ""
	published := false
	defer func() {
		if published || zipPath == originalZipPath {
			return
		}
		if info, statErr := os.Lstat(zipPath); statErr == nil && info.Mode().IsRegular() {
			_ = os.Remove(zipPath)
		}
	}()

	sk, err := s.skills.FindByID(nil, skillID)
	if err != nil {
		return ErrSkillNotFound
	}
	originalZipPath = filepath.Join(s.ZipDir(), filepath.Base(sk.ZipURL))
	if sk.AuthorID != userID {
		return ErrForbidden
	}
	if sk.Status == model.ResourceStatusPendingReview {
		return ErrSkillState
	}
	file, err := os.Open(zipPath)
	if err != nil {
		return platform.ErrInvalidInput
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return platform.ErrInvalidInput
	}
	skillMD, err := extractSkillMD(file, info.Size())
	if err != nil {
		return err
	}
	sk.ZipURL = zipURL
	sk.ZipFilename = zipFilename
	sk.FileSize = fileSize
	sk.SkillMD = skillMD
	if sk.Status == model.ResourceStatusPublished {
		sk.Status = model.ResourceStatusPendingReview
	}
	if err := s.skills.Update(nil, sk); err != nil {
		return err
	}
	published = true
	return nil
}

func (s *SkillService) Download(ctx context.Context, skillID uint) (string, error) {
	sk, err := s.skills.FindByID(nil, skillID)
	if err != nil || sk.Status != model.ResourceStatusPublished {
		return "", ErrSkillNotFound
	}
	if sk.ZipURL == "" {
		return "", ErrSkillState
	}
	if err := s.skills.IncrCount(nil, skillID, "downloads", 1); err != nil {
		return "", err
	}
	return sk.ZipURL, nil
}

func (s *SkillService) ToggleLike(ctx context.Context, userID, skillID uint) (bool, int, error) {
	sk, err := s.skills.FindByID(nil, skillID)
	if err != nil || sk.Status != model.ResourceStatusPublished {
		return false, 0, ErrSkillNotFound
	}
	var liked bool
	var newCount int
	err = s.skills.DB().Transaction(func(tx *gorm.DB) error {
		var err error
		liked, err = s.inter.ToggleSkillLike(tx, userID, skillID)
		if err != nil {
			return err
		}
		delta := 1
		if !liked {
			delta = -1
		}
		if err := s.skills.IncrCount(tx, skillID, "likes_count", delta); err != nil {
			return err
		}
		newCount = sk.LikesCount + delta
		return nil
	})
	if err == nil {
		go s.updateHotScoreAsync(skillID)
		if liked {
			go func() {
				_ = s.notifSvc.Create(context.Background(), sk.AuthorID, model.NotifTypeLikeSkill, "点赞", "有人赞了你的 Skill", "skill", skillID, userID)
			}()
		}
	}
	return liked, newCount, err
}

func (s *SkillService) ToggleFavorite(ctx context.Context, userID, skillID uint) (bool, int, error) {
	sk, err := s.skills.FindByID(nil, skillID)
	if err != nil || sk.Status != model.ResourceStatusPublished {
		return false, 0, ErrSkillNotFound
	}
	var favorited bool
	var newCount int
	err = s.skills.DB().Transaction(func(tx *gorm.DB) error {
		var err error
		favorited, err = s.inter.ToggleSkillFavorite(tx, userID, skillID)
		if err != nil {
			return err
		}
		delta := 1
		if !favorited {
			delta = -1
		}
		if err := s.skills.IncrCount(tx, skillID, "favorites_count", delta); err != nil {
			return err
		}
		newCount = sk.FavoritesCount + delta
		return nil
	})
	if err == nil {
		go s.updateHotScoreAsync(skillID)
	}
	return favorited, newCount, err
}

func (s *SkillService) updateHotScoreAsync(skillID uint) {
	sk, err := s.skills.FindByID(nil, skillID)
	if err != nil {
		return
	}
	score := CalculateHotScore(sk.Views, sk.LikesCount, sk.FavoritesCount, sk.CommentsCount, sk.CreatedAt, 1.5)
	_ = s.rdb.ZAdd(context.Background(), "rank:skills:hot", redis.Z{
		Score:  score,
		Member: skillID,
	}).Err()
}

func (s *SkillService) List(ctx context.Context, q SkillListQuery) (*SkillListResult, error) {
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
	case "hot", "downloads":
	default:
		q.Sort = "latest"
	}
	rq := repo.SkillQuery{
		Page: q.Page, PageSize: q.PageSize,
		TagID: q.TagID, Keyword: q.Keyword, AuthorID: q.AuthorID, Sort: q.Sort,
	}

	var key string
	if q.Sort == "hot" && q.TagID == nil && q.AuthorID == nil && q.Keyword == "" {
		key = fmt.Sprintf("hot:skills:%d:%d", q.Page, q.PageSize)
		if v, err := s.rdb.Get(ctx, key).Bytes(); err == nil {
			var res SkillListResult
			if json.Unmarshal(v, &res) == nil {
				return &res, nil
			}
		}
	}

	list, total, err := s.skills.List(ctx, rq)
	if err != nil {
		return nil, err
	}
	ids := make([]uint, 0, len(list))
	for _, sk := range list {
		ids = append(ids, sk.ID)
	}
	tagMap, err := s.skills.TagsForSkills(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := &SkillListResult{
		List:  make([]SkillSummary, 0, len(list)),
		Total: total, Page: q.Page, PageSize: q.PageSize,
	}
	for _, sk := range list {
		out.List = append(out.List, s.summaryOf(sk, tagMap[sk.ID]))
	}

	if key != "" {
		if b, err := json.Marshal(out); err == nil {
			_ = s.rdb.Set(ctx, key, b, s.cfg.HotCacheTTL).Err()
		}
	}
	return out, nil
}

func (s *SkillService) ListOwned(ctx context.Context, userID uint, status string, page, pageSize int) (*SkillListResult, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = s.cfg.DefaultPageSize
	}
	if pageSize > s.cfg.MaxPageSize {
		pageSize = s.cfg.MaxPageSize
	}
	list, total, err := s.skills.ListOwned(ctx, userID, status, page, pageSize)
	if err != nil {
		return nil, err
	}
	ids := make([]uint, 0, len(list))
	for _, skill := range list {
		ids = append(ids, skill.ID)
	}
	tagMap, err := s.skills.TagsForSkills(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := &SkillListResult{List: make([]SkillSummary, 0, len(list)), Total: total, Page: page, PageSize: pageSize}
	for _, skill := range list {
		out.List = append(out.List, s.summaryOf(skill, tagMap[skill.ID]))
	}
	return out, nil
}
