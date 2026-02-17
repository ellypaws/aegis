package sqlite

import (
	"errors"

	"gorm.io/gorm"

	"drigo/pkg/types"
)

// DefaultSettings returns the hardcoded defaults.
func DefaultSettings() types.Settings {
	return types.Settings{
		HeroTitle:       new("Tiered Vault"),
		HeroSubtitle:    new("How it works"),
		HeroDescription: new("Upload full + thumbnail. Gate access by Discord roles. Browse locked previews."),
		Theme: types.Theme{
			BorderRadius: "1.5rem",
			BorderSize:   "4px",

			// Light Defaults (Yellow/Zinc based on constants.ts)
			PrimaryColorLight:   "#fde047", // yellow-300
			SecondaryColorLight: "#fef08a", // yellow-200
			PageBgLight:         "#fef08a", // yellow-200
			PageBgTransLight:    1.0,
			CardBgLight:         "#ffffff",
			CardBgTransLight:    1.0,
			BorderColorLight:    "#fde047", // yellow-300

			// Dark Defaults
			PrimaryColorDark:   "#ca8a04", // yellow-600
			SecondaryColorDark: "#a16207", // yellow-700
			PageBgDark:         "#09090b", // zinc-950
			PageBgTransDark:    1.0,
			CardBgDark:         "#18181b", // zinc-900
			CardBgTransDark:    1.0,
			BorderColorDark:    "#ca8a04", // yellow-600
		},
	}
}

// GetSettings fetches the settings from the database.
// If no settings exist, it returns the defaults (but does NOT save them automatically unless we want to).
// For now, let's just return defaults if missing.
func (s *sqliteDB) GetSettings() (*types.Settings, error) {
	var settings types.Settings
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
func (s *sqliteDB) UpdateSettings(newSettings types.Settings) (*types.Settings, error) {
	var settings types.Settings
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
	settings.Theme = newSettings.Theme

	if err := s.db.Save(&settings).Error; err != nil {
		return nil, err
	}

	return &settings, nil
}
