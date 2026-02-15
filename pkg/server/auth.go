package server

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"

	"drigo/pkg/types"
)

// JwtCustomClaims are custom claims for JWT
type JwtCustomClaims struct {
	UserID   string            `json:"uid"`
	Username string            `json:"sub"`
	Avatar   string            `json:"avt"`
	Banner   string            `json:"ban"`
	IsAdmin  bool              `json:"adm"`
	Roles    []*discordgo.Role `json:"roles"`
	jwt.RegisteredClaims
}

func GetJWTSecret() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return []byte("secret")
	}
	return []byte(secret)
}

func GenerateToken(user *types.User, roles []*discordgo.Role) (string, error) {
	avatarURL := generateAvatarURL(user.UserID, user.Avatar)
	bannerURL := generateBannerURL(user.UserID, user.Banner)

	claims := &JwtCustomClaims{
		UserID:   user.UserID,
		Username: user.Username,
		Avatar:   avatarURL,
		Banner:   bannerURL,
		IsAdmin:  user.IsAdmin,
		Roles:    roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 72)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(GetJWTSecret())
}

func generateAvatarURL(userID, avatarHash string) string {
	if avatarHash == "" {
		// Generate default avatar URL based on user ID discriminator
		// Discord uses (user_id >> 22) % 6 for default avatar index
		discriminator := 0
		if len(userID) > 0 {
			for _, c := range userID {
				discriminator = (discriminator*10 + int(c-'0')) % 6
			}
		}
		return fmt.Sprintf("https://cdn.discordapp.com/embed/avatars/%d.png", discriminator)
	}

	if strings.HasPrefix(avatarHash, "a_") {
		return fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.gif", userID, avatarHash)
	}
	return fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png", userID, avatarHash)
}

func generateBannerURL(userID, bannerHash string) string {
	if bannerHash == "" {
		return ""
	}

	if strings.HasPrefix(bannerHash, "a_") {
		return fmt.Sprintf("https://cdn.discordapp.com/banners/%s/%s.gif", userID, bannerHash)
	}
	return fmt.Sprintf("https://cdn.discordapp.com/banners/%s/%s.png", userID, bannerHash)
}

// GetUserFromContext extracts claims from the echo context (set by middleware)
// Returns nil if no valid token
func GetUserFromContext(c echo.Context) *JwtCustomClaims {
	token, ok := c.Get("user").(*jwt.Token)
	if !ok {
		return nil
	}
	claims, ok := token.Claims.(*JwtCustomClaims)
	if !ok {
		return nil
	}
	return claims
}
