package repo

import (
	"context"

	"gorm.io/gorm"

	"aidevclub/internal/model"
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
	if err := r.exec(db).Preload("Category").Preload("Author").First(&a, id).Error; err != nil {
		return nil, err
	}
	return &a, nil
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
