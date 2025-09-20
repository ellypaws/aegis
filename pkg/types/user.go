package types

import (
	"github.com/bwmarrin/discordgo"
	"gorm.io/gorm"
)

// User is a minimal flattening of discordgo.User that can be losslessly
// reconstructed for common fields. (No JSON column required.)
type User struct {
	gorm.Model

	UserID        string `gorm:"uniqueIndex"` // discord ID
	Username      string
	GlobalName    string // new "display" name field in Discord
	Discriminator string // legacy, may be empty for new accounts
	Avatar        string
	Banner        string
	AccentColor   int
	Bot           bool
	System        bool
	PublicFlags   int // discordgo.UserFlags is an int
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
