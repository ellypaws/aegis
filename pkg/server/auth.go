package server

import (
	"os"
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
	claims := &JwtCustomClaims{
		UserID:   user.UserID,
		Username: user.Username,
		Avatar:   user.Avatar,
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
