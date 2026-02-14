package sqlite

import "drigo/pkg/types"

// ListPosts returns a list of posts, ordered by timestamp desc.
func (s *sqliteDB) ListPosts(limit, offset int) ([]*types.Post, error) {
	var posts []*types.Post
	err := s.db.
		Preload("Author").
		Preload("Image").
		Preload("Image.Blobs").
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
