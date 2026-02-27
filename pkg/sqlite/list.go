package sqlite

import (
	"drigo/pkg/types"

	"gorm.io/gorm"
)

// ListPosts returns a list of posts. sort can be "date" (by timestamp) or "id" (by insertion, default).
func (s *sqliteDB) ListPosts(limit, offset int, sort string) ([]*types.Post, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var posts []*types.Post

	orderClause := "id desc" // default: newest by insertion
	if sort == "date" {
		orderClause = "timestamp desc"
	}

	err := s.db.
		Preload("Author").
		Preload("Images", func(db *gorm.DB) *gorm.DB {
			return db.Select("id", "created_at", "updated_at", "deleted_at", "post_id", "(CASE WHEN length(thumbnail) > 0 THEN 1 ELSE 0 END) as has_thumbnail")
		}).
		Preload("Images.Blobs", func(db *gorm.DB) *gorm.DB {
			return db.Select("id", "created_at", "updated_at", "deleted_at", "image_id", "index", "content_type", "filename", "size")
		}).
		Preload("AllowedRoles").
		Order(orderClause).
		Limit(limit).
		Offset(offset).
		Find(&posts).Error
	if err != nil {
		return nil, err
	}
	return posts, nil
}
