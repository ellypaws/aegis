package types

import (
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

	Theme Theme `json:"theme" gorm:"embedded;embeddedPrefix:theme_"`
}

type Theme struct {
	// Global
	BorderRadius       string  `json:"border_radius"`       // e.g. "1.5rem"
	BorderSize         string  `json:"border_size"`         // e.g. "4px"
	GlobalTransparency float64 `json:"global_transparency"` // e.g. 0.95

	// Light Theme
	PrimaryColorLight    string  `json:"primary_color_light"`
	SecondaryColorLight  string  `json:"secondary_color_light"`
	PageBgLight          string  `json:"page_bg_light"`
	PageBgTransLight     float64 `json:"page_bg_trans_light"`
	PageBgImageLight     []byte  `json:"-" gorm:"type:blob"` // Raw bytes
	PageBgImageLightHash string  `json:"page_bg_image_light_hash"`
	CardBgLight          string  `json:"card_bg_light"`
	CardBgTransLight     float64 `json:"card_bg_trans_light"`
	BorderColorLight     string  `json:"border_color_light"`

	// Dark Theme
	PrimaryColorDark    string  `json:"primary_color_dark"`
	SecondaryColorDark  string  `json:"secondary_color_dark"`
	PageBgDark          string  `json:"page_bg_dark"`
	PageBgTransDark     float64 `json:"page_bg_trans_dark"`
	PageBgImageDark     []byte  `json:"-" gorm:"type:blob"` // Raw bytes
	PageBgImageDarkHash string  `json:"page_bg_image_dark_hash"`
	CardBgDark          string  `json:"card_bg_dark"`
	CardBgTransDark     float64 `json:"card_bg_trans_dark"`
	BorderColorDark     string  `json:"border_color_dark"`
}
