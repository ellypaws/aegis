package types

import (
	"time"

	"github.com/bwmarrin/discordgo"
)

// DomainPost is your in-memory convenience shape.
type DomainPost struct {
	ID        string
	ChannelID string
	GuildID   string
	Content   string
	Timestamp int64 // or time.Time if you prefer
	IsPremium bool
	Author    *discordgo.User
	Allowed   []discordgo.Role // or []*discordgo.Role
	Thumbnail []byte
	Images    [][]byte
}

func (p *Post) ToDomain() *DomainPost {
	dp := &DomainPost{
		ID:        p.PostKey,
		ChannelID: p.ChannelID,
		GuildID:   p.GuildID,
		Timestamp: p.Timestamp.Unix(),
		IsPremium: p.IsPremium,
	}
	// Author
	dp.Author = p.Author.ToDiscord()
	// Allowed roles (only lightweight fields we stored)
	if len(p.AllowedRoles) > 0 {
		dp.Allowed = make([]discordgo.Role, len(p.AllowedRoles))
		for i, a := range p.AllowedRoles {
			dp.Allowed[i] = discordgo.Role{
				ID:           a.RoleID,
				Name:         a.Name,
				Managed:      a.Managed,
				Mentionable:  a.Mentionable,
				Hoist:        a.Hoist,
				Color:        a.Color,
				Position:     a.Position,
				Permissions:  a.Permissions,
				Icon:         a.Icon,
				UnicodeEmoji: a.UnicodeEmoji,
				Flags:        discordgo.RoleFlags(a.Flags),
			}
		}
	}
	// Images
	if len(p.Images) > 0 {
		thumb, imgs := p.Images[0].GetImages()
		dp.Thumbnail = thumb
		dp.Images = imgs
	}

	return dp
}

func (p *Post) FromDomain(dp *DomainPost) {
	if dp == nil {
		return
	}
	p.PostKey = dp.ID
	p.ChannelID = dp.ChannelID
	p.GuildID = dp.GuildID
	p.Timestamp = timeFromUnix(dp.Timestamp)
	p.IsPremium = dp.IsPremium

	// Author
	if dp.Author != nil {
		var auth User
		auth.FromDiscord(dp.Author)
		p.Author = &auth
	}

	// Allowed: only store minimal flattened info you care about
	p.AllowedRoles = p.AllowedRoles[:0]
	for _, r := range dp.Allowed {
		p.AllowedRoles = append(p.AllowedRoles, Allowed{
			RoleID:       r.ID,
			Name:         r.Name,
			Managed:      r.Managed,
			Mentionable:  r.Mentionable,
			Hoist:        r.Hoist,
			Color:        r.Color,
			Position:     r.Position,
			Permissions:  r.Permissions,
			Icon:         r.Icon,
			UnicodeEmoji: r.UnicodeEmoji,
			Flags:        int(r.Flags),
		})
	}

	// Images
	if len(dp.Images) > 0 {
		img := Image{}
		img.SetImages(dp.Thumbnail, dp.Images)
		p.Images = []Image{img}
	}

}

func timeFromUnix(u int64) (t time.Time) {
	if u == 0 {
		return time.Time{}
	}
	return time.Unix(u, 0).UTC()
}
