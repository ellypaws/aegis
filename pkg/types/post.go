package types

import (
	"time"

	"gorm.io/gorm"
)

type Allowed struct {
	gorm.Model
	RoleID       string `gorm:"uniqueIndex;size:32" json:"id"` // Discord snowflake; add size for index efficiency
	Name         string `json:"name"`
	Managed      bool   `json:"managed"`
	Mentionable  bool   `json:"mentionable"`
	Hoist        bool   `json:"hoist"`
	Color        int    `json:"color"`
	Position     int    `json:"position"`
	Permissions  int64  `json:"permissions"`
	Icon         string `json:"icon"`
	UnicodeEmoji string `json:"unicodeEmoji"`
	Flags        int    `json:"flags"`
}

// Post stores message-like content with relations instead of JSON.
type Post struct {
	gorm.Model

	PostKey   string `gorm:"uniqueIndex;size:32" json:"postKey"` // external ID (ksuid)
	ChannelID string `gorm:"index;size:32" json:"channelId"`
	GuildID   string `gorm:"index;size:32" json:"guildId"`
	// Optional title and description shown in embeds
	Title       string    `gorm:"type:text" json:"title"`
	Description string    `gorm:"type:text" json:"description"`
	Timestamp   time.Time `gorm:"index" json:"timestamp"`
	IsPremium   bool      `json:"isPremium"`
	FocusX      *float64  `gorm:"default:50" json:"focusX"`
	FocusY      *float64  `gorm:"default:50" json:"focusY"`

	AuthorID uint  `gorm:"index;default:null" json:"authorId"` // FK to User (optional)
	Author   *User `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"author"`

	// Many-to-many; join table: post_allowed_roles
	AllowedRoles []Allowed `gorm:"many2many:post_allowed_roles" json:"allowedRoles"`

	// One-to-many images payload
	Images []Image `gorm:"constraint:OnDelete:CASCADE" json:"images"`
}
