package repo

import (
	"context"

	"gorm.io/gorm"

	"aidevclub/internal/model"
	"aidevclub/internal/platform"
)

type ArticleRepo struct{ db *gorm.DB }

func NewArticleRepo(db *gorm.DB) *ArticleRepo { return &ArticleRepo{db: db} }

func (r *ArticleRepo) DB() *gorm.DB { return r.db }

func (r *ArticleRepo) exec(db *gorm.DB) *gorm.DB {
	if db != nil {
		return db
	}
	return r.db
}

func (r *ArticleRepo) Create(db *gorm.DB, a *model.Article) error {
	return r.exec(db).Create(a).Error
}

func (r *ArticleRepo) FindByID(db *gorm.DB, id uint) (*model.Article, error) {
	var a model.Article
	if err := r.exec(db).Preload("Author").First(&a, id).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *ArticleRepo) FindByIDWithContext(ctx context.Context, id uint) (*model.Article, error) {
	return r.FindByID(r.db.WithContext(ctx), id)
}

func (r *ArticleRepo) Update(db *gorm.DB, a *model.Article) error {
	return r.exec(db).Save(a).Error
}

func (r *ArticleRepo) Delete(db *gorm.DB, id uint) error {
	return r.exec(db).Delete(&model.Article{}, id).Error
}

func (r *ArticleRepo) FindArticleTags(db *gorm.DB, articleID uint) ([]uint, error) {
	var rows []model.ArticleTag
	if err := r.exec(db).Where("article_id = ?", articleID).Find(&rows).Error; err != nil {
		return nil, err
	}
	ids := make([]uint, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.TagID)
	}
	return ids, nil
}

func (r *ArticleRepo) SetArticleTags(db *gorm.DB, articleID uint, tagIDs []uint) error {
	d := r.exec(db)
	if err := d.Where("article_id = ?", articleID).Delete(&model.ArticleTag{}).Error; err != nil {
		return err
	}
	if len(tagIDs) == 0 {
		return nil
	}
	rows := make([]model.ArticleTag, 0, len(tagIDs))
	for _, tid := range tagIDs {
		rows = append(rows, model.ArticleTag{ArticleID: articleID, TagID: tid})
	}
	return d.Create(&rows).Error
}

func (r *ArticleRepo) IncrViews(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Model(&model.Article{}).
		Where("id = ?", id).
		UpdateColumn("views", gorm.Expr("views + 1")).Error
}

func (r *ArticleRepo) IncrCount(db *gorm.DB, id uint, column string, delta int) error {
	return r.exec(db).Model(&model.Article{}).
		Where("id = ?", id).
		UpdateColumn(column, gorm.Expr(column+" + ?", delta)).Error
}

type ArticleQuery struct {
	Page, PageSize int
	TagID          *uint
	Keyword        string
	AuthorID       *uint
	Sort           string
}

func (r *ArticleRepo) baseQuery(ctx context.Context, q ArticleQuery) *gorm.DB {
	d := r.db.WithContext(ctx).Model(&model.Article{}).
		Where("status = ?", model.ArticleStatusPublished).
		Where("hidden = ?", false)
	if q.AuthorID != nil {
		d = d.Where("author_id = ?", *q.AuthorID)
	}
	if q.Keyword != "" {
		d = d.Where("MATCH(title, summary, content) AGAINST(? IN BOOLEAN MODE)", q.Keyword)
	}
	if q.TagID != nil {
		d = d.Where("id IN (SELECT article_id FROM article_tags WHERE tag_id = ?)", *q.TagID)
	}
	return d
}

func (r *ArticleRepo) Count(ctx context.Context, q ArticleQuery) (int64, error) {
	var total int64
	err := r.baseQuery(ctx, q).Count(&total).Error
	return total, err
}

func (r *ArticleRepo) List(ctx context.Context, q ArticleQuery) ([]model.Article, int64, error) {
	d := r.baseQuery(ctx, q)
	switch q.Sort {
	case "hot":
		d = d.Order("(views + 3*likes_count + 5*favorites_count + 2*comments_count) desc, id desc")
	case "pinned":
		d = d.Order("pinned desc, published_at desc, id desc")
	default:
		d = d.Order("published_at desc, id desc")
	}
	var list []model.Article
	if err := d.Preload("Author").
		Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	total, err := r.Count(ctx, q)
	return list, total, err
}

func (r *ArticleRepo) ListOwned(ctx context.Context, authorID uint, status string, page, pageSize int) ([]model.Article, int64, error) {
	if status != "" && status != string(model.ArticleStatusDraft) && status != string(model.ArticleStatusPublished) {
		return nil, 0, platform.ErrInvalidInput
	}
	d := r.db.WithContext(ctx).Model(&model.Article{}).Where("author_id = ?", authorID)
	if status != "" {
		d = d.Where("status = ?", status)
	}
	var total int64
	if err := d.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.Article
	if err := d.Order("updated_at desc, id desc").Preload("Author").
		Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *ArticleRepo) TagsForArticles(ctx context.Context, articleIDs []uint) (map[uint][]model.Tag, error) {
	res := map[uint][]model.Tag{}
	if len(articleIDs) == 0 {
		return res, nil
	}
	var rows []model.ArticleTag
	if err := r.db.WithContext(ctx).Where("article_id IN ?", articleIDs).Find(&rows).Error; err != nil {
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
		res[row.ArticleID] = append(res[row.ArticleID], byID[row.TagID])
	}
	return res, nil
}

type AdminArticleQuery struct {
	Keyword    string
	Visibility string
	AuthorID   *uint
	Page       int
	PageSize   int
}

func (r *ArticleRepo) AdminList(ctx context.Context, q AdminArticleQuery) ([]model.Article, int64, error) {
	d := r.db.WithContext(ctx).Model(&model.Article{}).
		Where("status = ?", model.ArticleStatusPublished)
	if q.Keyword != "" {
		like := "%" + q.Keyword + "%"
		d = d.Where("title LIKE ? OR summary LIKE ?", like, like)
	}
	if q.Visibility == "visible" {
		d = d.Where("hidden = ?", false)
	} else if q.Visibility == "hidden" {
		d = d.Where("hidden = ?", true)
	}
	if q.AuthorID != nil {
		d = d.Where("author_id = ?", *q.AuthorID)
	}
	var total int64
	if err := d.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []model.Article
	if err := d.Order("published_at desc, id desc").Preload("Author").
		Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *ArticleRepo) AdminFindByID(ctx context.Context, id uint) (*model.Article, error) {
	var a model.Article
	if err := r.db.WithContext(ctx).Preload("Author").First(&a, id).Error; err != nil {
		return nil, err
	}
	if a.Status != model.ArticleStatusPublished {
		return nil, gorm.ErrRecordNotFound
	}
	return &a, nil
}
