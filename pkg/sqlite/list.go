package sqlite

import (
	"drigo/pkg/types"

	"gorm.io/gorm"
)

// ListPosts returns a list of posts, ordered by timestamp desc.
func (s *sqliteDB) ListPosts(limit, offset int) ([]*types.Post, error) {
	var posts []*types.Post
	err := s.db.
		Preload("Author").
		Preload("Image", func(db *gorm.DB) *gorm.DB {
			return db.Select("id", "created_at", "updated_at", "deleted_at", "post_id", "(CASE WHEN length(thumbnail) > 0 THEN 1 ELSE 0 END) as has_thumbnail")
		}).
		Preload("Image.Blobs", func(db *gorm.DB) *gorm.DB {
			return db.Select("id", "created_at", "updated_at", "deleted_at", "image_id", "index", "content_type")
		}).
		Preload("AllowedRoles").
		Order("timestamp desc").
		Limit(limit).
		Offset(offset).
		Find(&posts).Error
	if err != nil {
		return nil, err
	}
	return posts, nil
}
