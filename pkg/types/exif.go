package types

import (
	"github.com/bwmarrin/discordgo"
	"github.com/charmbracelet/log"
)

type MemberExif struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name,omitempty"`
	Nitro       bool   `json:"nitro,omitempty"`
}

func ToMemberExif(member *discordgo.Member) *MemberExif {
	var memberExif *MemberExif
	if member != nil && member.User != nil {
		memberExif = &MemberExif{
			ID:          member.User.ID,
			Username:    member.User.Username,
			DisplayName: member.DisplayName(),
			Nitro:       member.User.PremiumType != 0,
		}
	} else {
		log.Warn("no member info available for exif")
	}
	return memberExif
}
