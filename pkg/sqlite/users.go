package sqlite

import (
	"errors"

	"gorm.io/gorm"

	"drigo/pkg/types"
)

func (s *sqliteDB) UserByID(id string) (*types.User, error) {
	var user types.User
	if err := s.db.Where("user_id = ?", id).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (s *sqliteDB) IsAdmin(id string) (bool, error) {
	user, err := s.UserByID(id)
	if err != nil {
		return false, err
	}
	if user == nil {
		return false, nil
	}
	return user.IsAdmin, nil
}

func (s *sqliteDB) UpsertUser(user *types.User) error {
	var existing types.User
	err := s.db.Where("user_id = ?", user.UserID).First(&existing).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// New user
		return s.db.Create(user).Error
	}

	// Update existing
	existing.Username = user.Username
	existing.Avatar = user.Avatar
	// Don't overwrite IsAdmin unless explicitly set?
	// The incoming user probably has IsAdmin=false if constructed from basic auth.
	// So we should NOT overwrite IsAdmin with false if it is already true in DB.
	// But `user` passed in might have partial data.

	// Actually, let's just update identity fields.
	return s.db.Model(&existing).Updates(map[string]interface{}{
		"username": user.Username,
		"avatar":   user.Avatar,
	}).Error
}

func (s *sqliteDB) CountUsers() (int64, error) {
	var count int64
	if err := s.db.Model(&types.User{}).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (s *sqliteDB) SetAdmin(id string, isAdmin bool) error {
	return s.db.Model(&types.User{}).Where("user_id = ?", id).Update("is_admin", isAdmin).Error
}

func (s *sqliteDB) GetAllUsers() ([]*types.User, error) {
	var users []*types.User
	if err := s.db.Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}
