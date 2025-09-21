package types

import (
	"time"

	"gorm.io/gorm"
)

// Allowed matches your original definition; unchanged, kept here for completeness.
type Allowed struct {
	gorm.Model
	RoleID       string `gorm:"uniqueIndex;size:32"` // Discord snowflake; add size for index efficiency
	Name         string
	Managed      bool
	Mentionable  bool
	Hoist        bool
	Color        int
	Position     int
	Permissions  int64
	Icon         string
	UnicodeEmoji string
	Flags        int
}

// Post stores message-like content with relations instead of JSON.
type Post struct {
	gorm.Model

	PostKey   string `gorm:"uniqueIndex;size:32"` // external ID (ksuid)
	ChannelID string `gorm:"index;size:32"`
	GuildID   string `gorm:"index;size:32"`
	// Optional title and description shown in embeds
	Title       string    `gorm:"type:text"`
	Description string    `gorm:"type:text"`
	Timestamp   time.Time `gorm:"index"`
	IsPremium   bool

	AuthorID uint  `gorm:"index;default:null"` // FK to User (optional)
	Author   *User `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`

	// Many-to-many; join table: post_allowed_roles
	AllowedRoles []Allowed `gorm:"many2many:post_allowed_roles"`

	// One-to-one image payload (with child blobs)
	Image *Image `gorm:"constraint:OnDelete:CASCADE"`
}
