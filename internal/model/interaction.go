package model

import "time"

type ArticleLike struct {
	ID        uint `gorm:"primaryKey"`
	ArticleID uint `gorm:"uniqueIndex:uniq_article_like;not null"`
	UserID    uint `gorm:"uniqueIndex:uniq_article_like;not null"`
	CreatedAt time.Time
}

type ArticleFavorite struct {
	ID        uint `gorm:"primaryKey"`
	ArticleID uint `gorm:"uniqueIndex:uniq_article_fav;not null"`
	UserID    uint `gorm:"uniqueIndex:uniq_article_fav;not null"`
	CreatedAt time.Time
}

type CommentLike struct {
	ID        uint `gorm:"primaryKey"`
	CommentID uint `gorm:"uniqueIndex:uniq_comment_like;not null"`
	UserID    uint `gorm:"uniqueIndex:uniq_comment_like;not null"`
	CreatedAt time.Time
}
