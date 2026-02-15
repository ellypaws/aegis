package sqlite

import (
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"drigo/pkg/types"
)

// CreatePost inserts a new post with its associations (Author, Image, AllowedRoles).
// It expects p.Author, p.Image, and p.AllowedRoles to be already set as desired.
func (s *sqliteDB) CreatePost(p *types.Post) error {
	if p == nil {
		return errors.New("nil post")
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		if p.Author != nil {

			var dbUser types.User
			q := tx.Where("user_id = ?", p.Author.UserID).First(&dbUser)
			if q.Error != nil {
				if errors.Is(q.Error, gorm.ErrRecordNotFound) {
					dbUser = types.User{}
				} else {
					return q.Error
				}
			}

			if dbUser.UserID == "" {
				dbUser.UserID = p.Author.UserID
			}
			dbUser.Username = p.Author.Username
			dbUser.GlobalName = p.Author.GlobalName
			dbUser.Discriminator = p.Author.Discriminator
			dbUser.Avatar = p.Author.Avatar
			dbUser.Banner = p.Author.Banner
			dbUser.AccentColor = p.Author.AccentColor
			dbUser.Bot = p.Author.Bot
			dbUser.System = p.Author.System
			dbUser.PublicFlags = p.Author.PublicFlags

			if dbUser.ID == 0 {
				if err := tx.Create(&dbUser).Error; err != nil {
					return err
				}
			} else {
				if err := tx.Save(&dbUser).Error; err != nil {
					return err
				}
			}
			p.AuthorID = dbUser.ID
			p.Author = &dbUser
		}

		if err := tx.Omit("Author", "Image", "AllowedRoles").Create(p).Error; err != nil {
			return err
		}

		// Image (optional) — attach to this post via PostID and create blobs explicitly
		if p.Image != nil {
			p.Image.PostID = p.ID
			// Prevent GORM from auto-creating blobs by omitting association
			blobs := p.Image.Blobs
			p.Image.Blobs = nil
			if err := tx.Omit("Blobs").Create(p.Image).Error; err != nil {
				return err
			}
			for i := range blobs {
				blobs[i].ImageID = p.Image.ID
			}
			if len(blobs) > 0 {
				if err := tx.Create(&blobs).Error; err != nil {
					return err
				}
			}
			// restore in-memory linkage
			p.Image.Blobs = blobs
		}

		// Allowed roles (optional). Ensure each role row exists; then attach.
		if len(p.AllowedRoles) > 0 {
			for i := range p.AllowedRoles {
				ar := &p.AllowedRoles[i]
				if ar.RoleID == "" {
					continue
				}
				if err := tx.Clauses(
					clause.OnConflict{
						Columns:   []clause.Column{{Name: "role_id"}},
						DoUpdates: clause.AssignmentColumns([]string{"name", "managed", "mentionable", "hoist", "color", "position", "permissions", "icon", "unicode_emoji", "flags"}),
					},
				).Create(ar).Error; err != nil {
					return err
				}
			}
			if err := tx.Model(p).Association("AllowedRoles").Replace(p.AllowedRoles); err != nil {
				return err
			}
		}

		return nil
	})
}

// ReadPost fetches a post by numeric PK with all associations preloaded.
func (s *sqliteDB) ReadPost(id uint) (*types.Post, error) {
	var p types.Post
	err := s.db.
		Preload("Author").
		Preload("Image", func(db *gorm.DB) *gorm.DB {
			return db.Select("id", "created_at", "updated_at", "deleted_at", "post_id", "(CASE WHEN length(thumbnail) > 0 THEN 1 ELSE 0 END) as has_thumbnail")
		}).
		Preload("Image.Blobs", func(db *gorm.DB) *gorm.DB {
			return db.Select("id", "created_at", "updated_at", "deleted_at", "image_id", "index", "content_type", "filename", "length(data) as size")
		}).
		Preload("AllowedRoles").
		First(&p, id).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// ReadPostByExternalID fetches by PostKey (external ID).
func (s *sqliteDB) ReadPostByExternalID(ext string) (*types.Post, error) {
	var p types.Post
	err := s.db.
		Preload("Author").
		Preload("Image", func(db *gorm.DB) *gorm.DB {
			return db.Select("id", "created_at", "updated_at", "deleted_at", "post_id", "(CASE WHEN length(thumbnail) > 0 THEN 1 ELSE 0 END) as has_thumbnail")
		}).
		Preload("Image.Blobs", func(db *gorm.DB) *gorm.DB {
			return db.Select("id", "created_at", "updated_at", "deleted_at", "image_id", "index", "content_type", "filename", "length(data) as size")
		}).
		Preload("AllowedRoles").
		Where("post_key = ?", ext).
		First(&p).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// UpdatePost replaces the post row and its associations to match p exactly.
// Use this when you want to overwrite AllowedRoles and Image in one go.
func (s *sqliteDB) UpdatePost(p *types.Post) error {
	if p == nil || p.ID == 0 {
		return errors.New("invalid post (nil or no ID)")
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		if p.Author != nil {
			var dbUser types.User
			q := tx.Where("user_id = ?", p.Author.UserID).First(&dbUser)
			if q.Error != nil {
				if errors.Is(q.Error, gorm.ErrRecordNotFound) {
					dbUser = types.User{}
				} else {
					return q.Error
				}
			}
			if dbUser.UserID == "" {
				dbUser.UserID = p.Author.UserID
			}
			dbUser.Username = p.Author.Username
			dbUser.GlobalName = p.Author.GlobalName
			dbUser.Discriminator = p.Author.Discriminator
			dbUser.Avatar = p.Author.Avatar
			dbUser.Banner = p.Author.Banner
			dbUser.AccentColor = p.Author.AccentColor
			dbUser.Bot = p.Author.Bot
			dbUser.System = p.Author.System
			dbUser.PublicFlags = p.Author.PublicFlags

			if dbUser.ID == 0 {
				if err := tx.Create(&dbUser).Error; err != nil {
					return err
				}
			} else {
				if err := tx.Save(&dbUser).Error; err != nil {
					return err
				}
			}
			p.AuthorID = dbUser.ID
			p.Author = &dbUser
		}

		// Image (optional) — ensure the image row belongs to this post and replace blobs
		if p.Image != nil {
			// Work on a local copy of blobs to avoid autosave
			blobs := p.Image.Blobs
			p.Image.Blobs = nil
			if p.Image.ID == 0 {
				p.Image.PostID = p.ID
				if err := tx.Omit("Blobs").Create(p.Image).Error; err != nil {
					return err
				}
			} else {
				p.Image.PostID = p.ID
				if err := tx.Omit("Blobs").Save(p.Image).Error; err != nil {
					return err
				}
			}
			if err := tx.Where("image_id = ?", p.Image.ID).Delete(&types.ImageBlob{}).Error; err != nil {
				return err
			}
			for i := range blobs {
				blobs[i].ImageID = p.Image.ID
			}
			if len(blobs) > 0 {
				if err := tx.Create(&blobs).Error; err != nil {
					return err
				}
			}
			p.Image.Blobs = blobs
		} else {
			// Remove any existing image for this post
			var imgs []types.Image
			if err := tx.Where("post_id = ?", p.ID).Find(&imgs).Error; err != nil {
				return err
			}
			for i := range imgs {
				img := imgs[i]
				if err := tx.Where("image_id = ?", img.ID).Delete(&types.ImageBlob{}).Error; err != nil {
					return err
				}
				if err := tx.Delete(&types.Image{}, img.ID).Error; err != nil {
					return err
				}
			}
		}

		// Update post core fields (omit associations, handled above)
		if err := tx.Omit("Author", "Image", "AllowedRoles").Save(p).Error; err != nil {
			return err
		}

		// Allowed roles (replace association)
		if err := tx.Model(p).Association("AllowedRoles").Clear(); err != nil {
			return err
		}
		if len(p.AllowedRoles) > 0 {
			for i := range p.AllowedRoles {
				ar := &p.AllowedRoles[i]
				if ar.RoleID == "" {
					continue
				}
				if err := tx.Clauses(
					clause.OnConflict{
						Columns:   []clause.Column{{Name: "role_id"}},
						DoUpdates: clause.AssignmentColumns([]string{"name", "managed", "mentionable", "hoist", "color", "position", "permissions", "icon", "unicode_emoji", "flags"}),
					},
				).Create(ar).Error; err != nil {
					return err
				}
			}
			if err := tx.Model(p).Association("AllowedRoles").Replace(p.AllowedRoles); err != nil {
				return err
			}
		}

		return nil
	})
}

// PostPatch updates selected scalar fields without touching associations unless specified.
type PostPatch struct {
	Content   *string
	Timestamp *time.Time
	IsPremium *bool
	ChannelID *string
	GuildID   *string

	// Optional association ops
	ReplaceAllowedRoles []*types.Allowed // if set, replaces AllowedRoles with these
	ClearAllowedRoles   bool             // if true, clears AllowedRoles

	ReplaceImage *types.Image // if non-nil, sets/replaces the Image (blobs fully replaced)
	ClearImage   bool         // if true, removes the Image
}

// PatchPost applies partial updates to a post by PK.
func (s *sqliteDB) PatchPost(id uint, patch PostPatch) error {
	if id == 0 {
		return errors.New("invalid id")
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		var p types.Post
		if err := tx.First(&p, id).Error; err != nil {
			return err
		}

		// Scalars
		updates := map[string]any{}
		if patch.Content != nil {
			updates["content"] = *patch.Content
		}
		if patch.Timestamp != nil {
			updates["timestamp"] = *patch.Timestamp
		}
		if patch.IsPremium != nil {
			updates["is_premium"] = *patch.IsPremium
		}
		if patch.ChannelID != nil {
			updates["channel_id"] = *patch.ChannelID
		}
		if patch.GuildID != nil {
			updates["guild_id"] = *patch.GuildID
		}
		if len(updates) > 0 {
			if err := tx.Model(&p).Updates(updates).Error; err != nil {
				return err
			}
		}

		// Allowed roles
		if patch.ClearAllowedRoles {
			if err := tx.Model(&p).Association("AllowedRoles").Clear(); err != nil {
				return err
			}
		}
		if patch.ReplaceAllowedRoles != nil {
			// ensure role rows exist/updated
			for i := range patch.ReplaceAllowedRoles {
				ar := patch.ReplaceAllowedRoles[i]
				if ar == nil || ar.RoleID == "" {
					continue
				}
				if err := tx.Clauses(
					clause.OnConflict{
						Columns:   []clause.Column{{Name: "role_id"}},
						DoUpdates: clause.AssignmentColumns([]string{"name", "managed", "mentionable", "hoist", "color", "position", "permissions", "icon", "unicode_emoji", "flags"}),
					},
				).Create(ar).Error; err != nil {
					return err
				}
			}
			if err := tx.Model(&p).Association("AllowedRoles").Replace(patch.ReplaceAllowedRoles); err != nil {
				return err
			}
		}

		// Image
		if patch.ClearImage {
			// delete image and child blobs if present for this post
			var imgs []types.Image
			if err := tx.Where("post_id = ?", p.ID).Find(&imgs).Error; err != nil {
				return err
			}
			for i := range imgs {
				img := imgs[i]
				if err := tx.Where("image_id = ?", img.ID).Delete(&types.ImageBlob{}).Error; err != nil {
					return err
				}
				if err := tx.Delete(&types.Image{}, img.ID).Error; err != nil {
					return err
				}
			}
		} else if patch.ReplaceImage != nil {
			// replace current image for this post with the provided one
			// first remove any existing images for this post
			if err := tx.Where("post_id = ?", p.ID).Delete(&types.Image{}).Error; err != nil {
				return err
			}
			img := patch.ReplaceImage
			img.PostID = p.ID
			blobs := img.Blobs
			img.Blobs = nil
			if err := tx.Omit("Blobs").Create(img).Error; err != nil {
				return err
			}
			for i := range blobs {
				blobs[i].ImageID = img.ID
			}
			if len(blobs) > 0 {
				if err := tx.Create(&blobs).Error; err != nil {
					return err
				}
			}
			img.Blobs = blobs
		}

		return nil
	})
}

// DeletePost removes the post and detaches associations.
// Image is deleted (with blobs) if referenced by this post.
func (s *sqliteDB) DeletePost(id uint) error {
	if id == 0 {
		return errors.New("invalid id")
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		var p types.Post
		if err := tx.Preload("AllowedRoles").First(&p, id).Error; err != nil {
			return err
		}

		// Clear m2m to drop join rows
		if err := tx.Model(&p).Association("AllowedRoles").Clear(); err != nil {
			return err
		}

		// Delete image + blobs if present
		var imgs []types.Image
		if err := tx.Where("post_id = ?", p.ID).Find(&imgs).Error; err != nil {
			return err
		}
		for i := range imgs {
			img := imgs[i]
			if err := tx.Where("image_id = ?", img.ID).Delete(&types.ImageBlob{}).Error; err != nil {
				return err
			}
			if err := tx.Delete(&types.Image{}, img.ID).Error; err != nil {
				return err
			}
		}

		// Finally delete post
		return tx.Delete(&types.Post{}, id).Error
	})
}

// GetImageBlob fetches a specific image blob by its ID including the data.
func (s *sqliteDB) GetImageBlob(id uint) (*types.ImageBlob, error) {
	var blob types.ImageBlob
	err := s.db.First(&blob, id).Error
	if err != nil {
		return nil, err
	}
	return &blob, nil
}

// GetImageThumbnailByBlobID fetches the thumbnail for the image associated with the given blob ID.
func (s *sqliteDB) GetImageThumbnailByBlobID(blobID uint) ([]byte, error) {
	var result struct {
		Thumbnail []byte
	}
	err := s.db.Table("images").
		Select("images.thumbnail").
		Joins("JOIN image_blobs ON image_blobs.image_id = images.id").
		Where("image_blobs.id = ?", blobID).
		Scan(&result).Error

	if err != nil {
		return nil, err
	}
	return result.Thumbnail, nil
}

// GetPostByBlobID finds the post associated with a specific image blob ID.
// Useful for permission checking before serving a blob.
func (s *sqliteDB) GetPostByBlobID(blobID uint) (*types.Post, error) {
	var blob types.ImageBlob
	if err := s.db.Select("image_id").First(&blob, blobID).Error; err != nil {
		return nil, err
	}

	var image types.Image
	if err := s.db.Select("post_id").First(&image, blob.ImageID).Error; err != nil {
		return nil, err
	}

	// ReadPost logic but without preloading blobs data
	var p types.Post
	err := s.db.
		Preload("Author").
		Preload("AllowedRoles").
		First(&p, image.PostID).Error
	if err != nil {
		return nil, err
	}

	return &p, nil
}
