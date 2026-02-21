package sqlite

import (
	"gorm.io/gorm/clause"

	"drigo/pkg/types"
)

func (s *sqliteDB) UpdateCachedRoles(roles []types.CachedRole) error {
	if len(roles) == 0 {
		return nil
	}
	// Use clauses.OnConflict to perform an upsert on ID
	return s.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"name", "color", "managed"}),
	}).Create(&roles).Error
}

func (s *sqliteDB) UpdateCachedChannels(channels []types.CachedChannel) error {
	if len(channels) == 0 {
		return nil
	}
	return s.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"name", "type"}),
	}).Create(&channels).Error
}

func (s *sqliteDB) GetCachedRoles() ([]types.CachedRole, error) {
	var roles []types.CachedRole
	err := s.db.Find(&roles).Error
	return roles, err
}

func (s *sqliteDB) GetCachedChannels() ([]types.CachedChannel, error) {
	var channels []types.CachedChannel
	err := s.db.Find(&channels).Error
	return channels, err
}
