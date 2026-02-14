package types

import (
	"gorm.io/gorm"

	"github.com/bwmarrin/discordgo"
)

// User is a minimal flattening of discordgo.User that can be losslessly
// reconstructed for common fields. (No JSON column required.)
type User struct {
	gorm.Model

	UserID        string `gorm:"uniqueIndex" json:"userId"` // discord ID
	Username      string `json:"username"`
	GlobalName    string `json:"globalName"`    // new "display" name field in Discord
	Discriminator string `json:"discriminator"` // legacy, may be empty for new accounts
	Avatar        string `json:"avatar"`
	Banner        string `json:"banner"`
	AccentColor   int    `json:"accentColor"`
	Bot           bool   `json:"bot"`
	System        bool   `json:"system"`
	PublicFlags   int    `json:"publicFlags"` // discordgo.UserFlags is an int

	IsAdmin bool `json:"isAdmin"`
}

func (u *User) ToDiscord() *discordgo.User {
	if u == nil {
		return nil
	}
	return &discordgo.User{
		ID:            u.UserID,
		Username:      u.Username,
		GlobalName:    u.GlobalName,
		Discriminator: u.Discriminator,
		Avatar:        u.Avatar,
		Banner:        u.Banner,
		AccentColor:   u.AccentColor,
		Bot:           u.Bot,
		System:        u.System,
		PublicFlags:   discordgo.UserFlags(u.PublicFlags),
	}
}

func (u *User) FromDiscord(d *discordgo.User) {
	if d == nil {
		return
	}
	u.UserID = d.ID
	u.Username = d.Username
	u.GlobalName = d.GlobalName
	u.Discriminator = d.Discriminator
	u.Avatar = d.Avatar
	u.Banner = d.Banner
	u.AccentColor = d.AccentColor
	u.Bot = d.Bot
	u.System = d.System
	u.PublicFlags = int(d.PublicFlags)
}
