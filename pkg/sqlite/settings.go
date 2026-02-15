package sqlite

import (
	"errors"

	"gorm.io/gorm"
)

// Settings represents the global application settings.
// We only expect one row in this table.
type Settings struct {
	gorm.Model
	// Hero / Home Page Text
	HeroTitle       *string `json:"hero_title"`       // e.g. "Tiered Vault"
	HeroSubtitle    *string `json:"hero_subtitle"`    // e.g. "How it works"
	HeroDescription *string `json:"hero_description"` // e.g. "Upload full + thumbnail..."
	// Button Text? Or other UI strings can be added here.
}

// DefaultSettings returns the hardcoded defaults.
func DefaultSettings() Settings {
	return Settings{
		HeroTitle:       new("Tiered Vault"),
		HeroSubtitle:    new("How it works"),
		HeroDescription: new("Upload full + thumbnail. Gate access by Discord roles. Browse locked previews."),
	}
}

// GetSettings fetches the settings from the database.
// If no settings exist, it returns the defaults (but does NOT save them automatically unless we want to).
// For now, let's just return defaults if missing.
func (s *sqliteDB) GetSettings() (*Settings, error) {
	var settings Settings
	err := s.db.First(&settings).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			defaults := DefaultSettings()
			return &defaults, nil
		}
		return nil, err
	}
	return &settings, nil
}

// UpdateSettings updates the existing settings or creates specific ones.
// We strictly maintain only one row (ID=1) or just the first row.
func (s *sqliteDB) UpdateSettings(newSettings Settings) (*Settings, error) {
	var settings Settings
	err := s.db.First(&settings).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Create new
		settings = newSettings
		// Ensure we don't accidentally write a huge ID or something if GORM assumes auto-increment
		// We can just let GORM handle ID 1.
		if err := s.db.Create(&settings).Error; err != nil {
			return nil, err
		}
		return &settings, nil
	} else if err != nil {
		return nil, err
	}

	// Update existing fields
	settings.HeroTitle = newSettings.HeroTitle
	settings.HeroSubtitle = newSettings.HeroSubtitle
	settings.HeroDescription = newSettings.HeroDescription

	if err := s.db.Save(&settings).Error; err != nil {
		return nil, err
	}

	return &settings, nil
}
