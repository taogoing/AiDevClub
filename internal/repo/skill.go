package repo

import (
	"context"

	"gorm.io/gorm"

	"aidevclub/internal/model"
	"aidevclub/internal/platform"
)

type SkillRepo struct{ db *gorm.DB }

func NewSkillRepo(db *gorm.DB) *SkillRepo { return &SkillRepo{db: db} }

func (r *SkillRepo) DB() *gorm.DB { return r.db }

func (r *SkillRepo) exec(db *gorm.DB) *gorm.DB {
	if db != nil {
		return db
	}
	return r.db
}

func (r *SkillRepo) Create(db *gorm.DB, s *model.Skill) error {
	return r.exec(db).Create(s).Error
}

func (r *SkillRepo) FindByID(db *gorm.DB, id uint) (*model.Skill, error) {
	var s model.Skill
	if err := r.exec(db).Preload("Author").First(&s, id).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *SkillRepo) FindByIDWithContext(ctx context.Context, id uint) (*model.Skill, error) {
	return r.FindByID(r.db.WithContext(ctx), id)
}

func (r *SkillRepo) Update(db *gorm.DB, s *model.Skill) error {
	return r.exec(db).Save(s).Error
}

func (r *SkillRepo) Delete(db *gorm.DB, id uint) error {
	return r.exec(db).Delete(&model.Skill{}, id).Error
}

func (r *SkillRepo) FindSkillTags(db *gorm.DB, skillID uint) ([]uint, error) {
	var rows []model.SkillTag
	if err := r.exec(db).Where("skill_id = ?", skillID).Find(&rows).Error; err != nil {
		return nil, err
	}
	ids := make([]uint, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.TagID)
	}
	return ids, nil
}

func (r *SkillRepo) SetSkillTags(db *gorm.DB, skillID uint, tagIDs []uint) error {
	d := r.exec(db)
	if err := d.Where("skill_id = ?", skillID).Delete(&model.SkillTag{}).Error; err != nil {
		return err
	}
	if len(tagIDs) == 0 {
		return nil
	}
	rows := make([]model.SkillTag, 0, len(tagIDs))
	for _, tid := range tagIDs {
		rows = append(rows, model.SkillTag{SkillID: skillID, TagID: tid})
	}
	return d.Create(&rows).Error
}

func (r *SkillRepo) IncrViews(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Model(&model.Skill{}).
		Where("id = ?", id).
		UpdateColumn("views", gorm.Expr("views + 1")).Error
}

func (r *SkillRepo) IncrCount(db *gorm.DB, id uint, column string, delta int) error {
	return r.exec(db).Model(&model.Skill{}).
		Where("id = ?", id).
		UpdateColumn(column, gorm.Expr(column+" + ?", delta)).Error
}

type SkillQuery struct {
	Page, PageSize int
	TagID          *uint
	Keyword        string
	AuthorID       *uint
	Sort           string
}

func (r *SkillRepo) baseQuery(ctx context.Context, q SkillQuery) *gorm.DB {
	d := r.db.WithContext(ctx).Model(&model.Skill{}).
		Where("status = ?", model.ResourceStatusPublished).
		Where("hidden = ?", false)
	if q.AuthorID != nil {
		d = d.Where("author_id = ?", *q.AuthorID)
	}
	if q.Keyword != "" {
		d = d.Where("MATCH(name, description) AGAINST(? IN BOOLEAN MODE)", q.Keyword)
	}
	if q.TagID != nil {
		d = d.Where("id IN (SELECT skill_id FROM skill_tags WHERE tag_id = ?)", *q.TagID)
	}
	return d
}

func (r *SkillRepo) Count(ctx context.Context, q SkillQuery) (int64, error) {
	var total int64
	err := r.baseQuery(ctx, q).Count(&total).Error
	return total, err
}

func (r *SkillRepo) List(ctx context.Context, q SkillQuery) ([]model.Skill, int64, error) {
	d := r.baseQuery(ctx, q)
	switch q.Sort {
	case "hot":
		d = d.Order("(views + 3*likes_count + 5*favorites_count + 2*comments_count) desc, id desc")
	default:
		d = d.Order("published_at desc, id desc")
	}
	var list []model.Skill
	if err := d.Preload("Author").
		Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	total, err := r.Count(ctx, q)
	return list, total, err
}

func (r *SkillRepo) ListOwned(ctx context.Context, authorID uint, status string, page, pageSize int) ([]model.Skill, int64, error) {
	if status != "" && !validResourceStatus(status) {
		return nil, 0, platform.ErrInvalidInput
	}
	d := r.db.WithContext(ctx).Model(&model.Skill{}).Where("author_id = ?", authorID)
	if status != "" {
		d = d.Where("status = ?", status)
	}
	var total int64
	if err := d.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.Skill
	if err := d.Order("updated_at desc, id desc").Preload("Author").
		Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func validResourceStatus(status string) bool {
	switch model.ResourceStatus(status) {
	case model.ResourceStatusDraft, model.ResourceStatusPendingReview, model.ResourceStatusPublished, model.ResourceStatusRejected, model.ResourceStatusArchived:
		return true
	default:
		return false
	}
}

func (r *SkillRepo) CountByStatus(ctx context.Context, status model.ResourceStatus) (int64, error) {
	var total int64
	err := r.db.WithContext(ctx).Model(&model.Skill{}).Where("status = ?", status).Count(&total).Error
	return total, err
}

func (r *SkillRepo) TagsForSkills(ctx context.Context, skillIDs []uint) (map[uint][]model.Tag, error) {
	res := map[uint][]model.Tag{}
	if len(skillIDs) == 0 {
		return res, nil
	}
	var rows []model.SkillTag
	if err := r.db.WithContext(ctx).Where("skill_id IN ?", skillIDs).Find(&rows).Error; err != nil {
		return nil, err
	}
	tagIDs := map[uint]bool{}
	for _, row := range rows {
		tagIDs[row.TagID] = true
	}
	ids := make([]uint, 0, len(tagIDs))
	for id := range tagIDs {
		ids = append(ids, id)
	}
	var tags []model.Tag
	if len(ids) > 0 {
		if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&tags).Error; err != nil {
			return nil, err
		}
	}
	byID := map[uint]model.Tag{}
	for _, t := range tags {
		byID[t.ID] = t
	}
	for _, row := range rows {
		res[row.SkillID] = append(res[row.SkillID], byID[row.TagID])
	}
	return res, nil
}

type AdminResourceQuery struct {
	Keyword  string
	Status   string
	AuthorID *uint
	TagID    *uint
	Page     int
	PageSize int
}

func (r *SkillRepo) AdminList(ctx context.Context, q AdminResourceQuery) ([]model.Skill, int64, error) {
	d := r.db.WithContext(ctx).Model(&model.Skill{})
	if q.Status != "" {
		d = d.Where("status = ?", q.Status)
	} else {
		d = d.Where("status = ?", model.ResourceStatusPendingReview)
	}
	if q.Keyword != "" {
		like := "%" + q.Keyword + "%"
		d = d.Where("name LIKE ? OR description LIKE ?", like, like)
	}
	if q.AuthorID != nil {
		d = d.Where("author_id = ?", *q.AuthorID)
	}
	if q.TagID != nil {
		d = d.Where("id IN (SELECT skill_id FROM skill_tags WHERE tag_id = ?)", *q.TagID)
	}
	var total int64
	if err := d.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.Skill
	if err := d.Order("updated_at desc, id desc").Preload("Author").
		Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *SkillRepo) AdminFindByID(ctx context.Context, id uint) (*model.Skill, error) {
	var s model.Skill
	if err := r.db.WithContext(ctx).Preload("Author").First(&s, id).Error; err != nil {
		return nil, err
	}
	return &s, nil
}
