package repo

import (
	"context"

	"gorm.io/gorm"

	"aidevclub/internal/model"
)

type McpServerRepo struct{ db *gorm.DB }

func NewMcpServerRepo(db *gorm.DB) *McpServerRepo { return &McpServerRepo{db: db} }

func (r *McpServerRepo) DB() *gorm.DB { return r.db }

func (r *McpServerRepo) exec(db *gorm.DB) *gorm.DB {
	if db != nil {
		return db
	}
	return r.db
}

func (r *McpServerRepo) Create(db *gorm.DB, s *model.McpServer) error {
	return r.exec(db).Create(s).Error
}

func (r *McpServerRepo) FindByID(db *gorm.DB, id uint) (*model.McpServer, error) {
	var s model.McpServer
	if err := r.exec(db).Preload("Author").First(&s, id).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *McpServerRepo) Update(db *gorm.DB, s *model.McpServer) error {
	return r.exec(db).Save(s).Error
}

func (r *McpServerRepo) Delete(db *gorm.DB, id uint) error {
	return r.exec(db).Delete(&model.McpServer{}, id).Error
}

func (r *McpServerRepo) FindMcpServerTags(db *gorm.DB, serverID uint) ([]uint, error) {
	var rows []model.McpServerTag
	if err := r.exec(db).Where("mcp_server_id = ?", serverID).Find(&rows).Error; err != nil {
		return nil, err
	}
	ids := make([]uint, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.TagID)
	}
	return ids, nil
}

func (r *McpServerRepo) SetMcpServerTags(db *gorm.DB, serverID uint, tagIDs []uint) error {
	d := r.exec(db)
	if err := d.Where("mcp_server_id = ?", serverID).Delete(&model.McpServerTag{}).Error; err != nil {
		return err
	}
	if len(tagIDs) == 0 {
		return nil
	}
	rows := make([]model.McpServerTag, 0, len(tagIDs))
	for _, tid := range tagIDs {
		rows = append(rows, model.McpServerTag{McpServerID: serverID, TagID: tid})
	}
	return d.Create(&rows).Error
}

func (r *McpServerRepo) IncrViews(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Model(&model.McpServer{}).
		Where("id = ?", id).
		UpdateColumn("views", gorm.Expr("views + 1")).Error
}

func (r *McpServerRepo) IncrCount(db *gorm.DB, id uint, column string, delta int) error {
	return r.exec(db).Model(&model.McpServer{}).
		Where("id = ?", id).
		UpdateColumn(column, gorm.Expr(column+" + ?", delta)).Error
}

type McpServerQuery struct {
	Page, PageSize int
	TagID          *uint
	Keyword        string
	AuthorID       *uint
	Sort           string
}

func (r *McpServerRepo) baseQuery(ctx context.Context, q McpServerQuery) *gorm.DB {
	d := r.db.WithContext(ctx).Model(&model.McpServer{}).Where("status = ?", model.ResourceStatusPublished)
	if q.AuthorID != nil {
		d = d.Where("author_id = ?", *q.AuthorID)
	}
	if q.Keyword != "" {
		kw := "%" + q.Keyword + "%"
		d = d.Where("(name LIKE ? OR description LIKE ?)", kw, kw)
	}
	if q.TagID != nil {
		d = d.Where("id IN (SELECT mcp_server_id FROM mcp_server_tags WHERE tag_id = ?)", *q.TagID)
	}
	return d
}

func (r *McpServerRepo) Count(ctx context.Context, q McpServerQuery) (int64, error) {
	var total int64
	err := r.baseQuery(ctx, q).Count(&total).Error
	return total, err
}

func (r *McpServerRepo) List(ctx context.Context, q McpServerQuery) ([]model.McpServer, int64, error) {
	d := r.baseQuery(ctx, q)
	switch q.Sort {
	case "hot":
		d = d.Order("(views + 3*likes_count + 5*favorites_count + 2*comments_count) desc, id desc")
	case "downloads":
		d = d.Order("downloads desc, id desc")
	default:
		d = d.Order("published_at desc, id desc")
	}
	var list []model.McpServer
	if err := d.Preload("Author").
		Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	total, err := r.Count(ctx, q)
	return list, total, err
}

func (r *McpServerRepo) TagsForMcpServers(ctx context.Context, serverIDs []uint) (map[uint][]model.Tag, error) {
	res := map[uint][]model.Tag{}
	if len(serverIDs) == 0 {
		return res, nil
	}
	var rows []model.McpServerTag
	if err := r.db.WithContext(ctx).Where("mcp_server_id IN ?", serverIDs).Find(&rows).Error; err != nil {
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
		res[row.McpServerID] = append(res[row.McpServerID], byID[row.TagID])
	}
	return res, nil
}
