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
	s.mu.Lock()
	defer s.mu.Unlock()
	if p == nil {
		return errors.New("nil post")
	}
	if p.Timestamp.IsZero() {
		p.Timestamp = time.Now().UTC()
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

		if err := tx.Omit("Author", "Images", "AllowedRoles").Create(p).Error; err != nil {
			return err
		}

		// Images (optional) — attach to this post via PostID and create blobs explicitly
		if len(p.Images) > 0 {
			for i := range p.Images {
				p.Images[i].ID = 0
				p.Images[i].PostID = p.ID
				// Prevent GORM from auto-creating blobs by omitting association
				blobs := p.Images[i].Blobs
				p.Images[i].Blobs = nil
				if err := tx.Omit("Blobs").Create(&p.Images[i]).Error; err != nil {
					return err
				}
				for j := range blobs {
					blobs[j].ID = 0
					blobs[j].ImageID = p.Images[i].ID
				}
				if len(blobs) > 0 {
					if err := tx.Create(&blobs).Error; err != nil {
						return err
					}
				}
				// restore in-memory linkage
				p.Images[i].Blobs = blobs
			}
		}

		// Allowed roles (optional). Ensure each role row exists; then attach.
		if len(p.AllowedRoles) > 0 {
			resolvedRoles := make([]types.Allowed, 0, len(p.AllowedRoles))
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

				var resolved types.Allowed
				if err := tx.Where("role_id = ?", ar.RoleID).First(&resolved).Error; err != nil {
					return err
				}
				resolvedRoles = append(resolvedRoles, resolved)
			}
			p.AllowedRoles = resolvedRoles
			if err := tx.Model(p).Association("AllowedRoles").Replace(p.AllowedRoles); err != nil {
				return err
			}
		}

		return nil
	})
}

// ReadPost fetches a post by numeric PK with all associations preloaded.
func (s *sqliteDB) ReadPost(id uint) (*types.Post, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var p types.Post
	err := s.db.
		Preload("Author").
		Preload("Author").
		Preload("Images", func(db *gorm.DB) *gorm.DB {
			return db.Select("id", "created_at", "updated_at", "deleted_at", "post_id", "(CASE WHEN length(thumbnail) > 0 THEN 1 ELSE 0 END) as has_thumbnail")
		}).
		Preload("Images.Blobs", func(db *gorm.DB) *gorm.DB {
			return db.Select("id", "created_at", "updated_at", "deleted_at", "image_id", "index", "content_type", "filename", "length(data) as size")
		}).
		Preload("AllowedRoles").
		First(&p, id).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// ReadPostWithBlobData fetches a post by numeric PK with full image/blob payload.
// Use this for edit flows that must preserve existing binary data.
func (s *sqliteDB) ReadPostWithBlobData(id uint) (*types.Post, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var p types.Post
	err := s.db.
		Preload("Author").
		Preload("Images").
		Preload("Images.Blobs", func(db *gorm.DB) *gorm.DB {
			return db.Order("`image_blobs`.`index` ASC")
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
	s.mu.RLock()
	defer s.mu.RUnlock()
	var p types.Post
	err := s.db.
		Preload("Author").
		Preload("Author").
		Preload("Images", func(db *gorm.DB) *gorm.DB {
			return db.Select("id", "created_at", "updated_at", "deleted_at", "post_id", "(CASE WHEN length(thumbnail) > 0 THEN 1 ELSE 0 END) as has_thumbnail")
		}).
		Preload("Images.Blobs", func(db *gorm.DB) *gorm.DB {
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

// ReadPostByExternalIDWithBlobData fetches by PostKey (external ID) with full image/blob payload.
// Use this for edit flows that must preserve existing binary data.
func (s *sqliteDB) ReadPostByExternalIDWithBlobData(ext string) (*types.Post, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var p types.Post
	err := s.db.
		Preload("Author").
		Preload("Images").
		Preload("Images.Blobs", func(db *gorm.DB) *gorm.DB {
			return db.Order("`image_blobs`.`index` ASC")
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
	s.mu.Lock()
	defer s.mu.Unlock()
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

		// Images (optional) — ensure the image rows belong to this post and replace blobs
		if len(p.Images) > 0 {
			// First, remove all existing images for this post to handle replacements cleanly
			// (Optimization: could diff IDs but full replacement is safer and simpler for execution)
			var existingImgs []types.Image
			if err := tx.Where("post_id = ?", p.ID).Find(&existingImgs).Error; err != nil {
				return err
			}
			for i := range existingImgs {
				img := existingImgs[i]
				if err := tx.Unscoped().Where("image_id = ?", img.ID).Delete(&types.ImageBlob{}).Error; err != nil {
					return err
				}
				if err := tx.Unscoped().Delete(&types.Image{}, img.ID).Error; err != nil {
					return err
				}
			}

			// Add new images
			for i := range p.Images {
				img := &p.Images[i]
				img.ID = 0
				img.PostID = p.ID
				blobs := img.Blobs
				img.Blobs = nil
				if err := tx.Omit("Blobs").Create(img).Error; err != nil {
					return err
				}
				for j := range blobs {
					blobs[j].ImageID = img.ID
					blobs[j].ID = 0 // ensure fresh insert
				}
				if len(blobs) > 0 {
					if err := tx.Create(&blobs).Error; err != nil {
						return err
					}
				}
				img.Blobs = blobs
			}
		} else {
			// Remove any existing image for this post
			var imgs []types.Image
			if err := tx.Where("post_id = ?", p.ID).Find(&imgs).Error; err != nil {
				return err
			}
			for i := range imgs {
				img := imgs[i]
				if err := tx.Unscoped().Where("image_id = ?", img.ID).Delete(&types.ImageBlob{}).Error; err != nil {
					return err
				}
				if err := tx.Unscoped().Delete(&types.Image{}, img.ID).Error; err != nil {
					return err
				}
			}
		}

		// Update post core fields (omit associations, handled above)
		if err := tx.Omit("Author", "Images", "AllowedRoles").Save(p).Error; err != nil {
			return err
		}

		// Allowed roles (replace association)
		rolesToApply := append([]types.Allowed(nil), p.AllowedRoles...)
		if err := tx.Model(p).Association("AllowedRoles").Clear(); err != nil {
			return err
		}
		if len(rolesToApply) > 0 {
			resolvedRoles := make([]types.Allowed, 0, len(rolesToApply))
			for i := range rolesToApply {
				ar := &rolesToApply[i]
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

				var resolved types.Allowed
				if err := tx.Where("role_id = ?", ar.RoleID).First(&resolved).Error; err != nil {
					return err
				}
				resolvedRoles = append(resolvedRoles, resolved)
			}
			if err := tx.Model(p).Association("AllowedRoles").Replace(resolvedRoles); err != nil {
				return err
			}
			p.AllowedRoles = resolvedRoles
		} else {
			p.AllowedRoles = nil
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

	ReplaceImages []types.Image // if non-nil, sets/replaces the Images (blobs fully replaced)
	ClearImages   bool          // if true, removes the Images
}

// PatchPost applies partial updates to a post by PK.
func (s *sqliteDB) PatchPost(id uint, patch PostPatch) error {
	s.mu.Lock()
	defer s.mu.Unlock()
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

		// Images
		if patch.ClearImages {
			// delete image and child blobs if present for this post
			var imgs []types.Image
			if err := tx.Where("post_id = ?", p.ID).Find(&imgs).Error; err != nil {
				return err
			}
			for i := range imgs {
				img := imgs[i]
				if err := tx.Unscoped().Where("image_id = ?", img.ID).Delete(&types.ImageBlob{}).Error; err != nil {
					return err
				}
				if err := tx.Unscoped().Delete(&types.Image{}, img.ID).Error; err != nil {
					return err
				}
			}
		} else if patch.ReplaceImages != nil {
			// replace current images for this post with the provided ones
			// first remove any existing images for this post
			var existingImgs []types.Image
			if err := tx.Where("post_id = ?", p.ID).Find(&existingImgs).Error; err != nil {
				return err
			}
			for i := range existingImgs {
				img := existingImgs[i]
				if err := tx.Unscoped().Where("image_id = ?", img.ID).Delete(&types.ImageBlob{}).Error; err != nil {
					return err
				}
				if err := tx.Unscoped().Delete(&types.Image{}, img.ID).Error; err != nil {
					return err
				}
			}
			for _, img := range patch.ReplaceImages {
				img.ID = 0
				img.PostID = p.ID
				blobs := img.Blobs
				img.Blobs = nil
				if err := tx.Omit("Blobs").Create(&img).Error; err != nil {
					return err
				}
				for i := range blobs {
					blobs[i].ID = 0
					blobs[i].ImageID = img.ID
				}
				if len(blobs) > 0 {
					if err := tx.Create(&blobs).Error; err != nil {
						return err
					}
				}
				// img.Blobs = blobs // (loop variable, no need to update back)
			}
		}

		return nil
	})
}

// DeletePost removes the post and detaches associations.
// Image is deleted (with blobs) if referenced by this post.
func (s *sqliteDB) DeletePost(id uint) error {
	s.mu.Lock()
	defer s.mu.Unlock()
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
	s.mu.RLock()
	defer s.mu.RUnlock()
	var blob types.ImageBlob
	err := s.db.First(&blob, id).Error
	if err != nil {
		return nil, err
	}
	return &blob, nil
}

// GetImageThumbnailByBlobID fetches the thumbnail for the image associated with the given blob ID.
func (s *sqliteDB) GetImageThumbnailByBlobID(blobID uint) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
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
	s.mu.RLock()
	defer s.mu.RUnlock()
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
