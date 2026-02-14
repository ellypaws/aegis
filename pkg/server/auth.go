package server

import (
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"

	"drigo/pkg/types"
)

// JwtCustomClaims are custom claims for JWT
type JwtCustomClaims struct {
	UserID   string   `json:"uid"`
	Username string   `json:"sub"`
	IsAdmin  bool     `json:"adm"`
	RoleIDs  []string `json:"roles"`
	jwt.RegisteredClaims
}

func GetJWTSecret() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return []byte("secret")
	}
	return []byte(secret)
}

func GenerateToken(user *types.User, roleIDs []string) (string, error) {
	claims := &JwtCustomClaims{
		UserID:   user.UserID,
		Username: user.Username,
		IsAdmin:  user.IsAdmin,
		RoleIDs:  roleIDs,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 72)), // 3 days
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
