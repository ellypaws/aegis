package server

import (
	"fmt"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/charmbracelet/log"
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

type SessionUser struct {
	UserID        string            `json:"userId"`
	Username      string            `json:"username"`
	GlobalName    string            `json:"globalName"`
	Discriminator string            `json:"discriminator"`
	Avatar        string            `json:"avatar"`
	Banner        string            `json:"banner"`
	AccentColor   int               `json:"accentColor"`
	Bot           bool              `json:"bot"`
	System        bool              `json:"system"`
	PublicFlags   int               `json:"publicFlags"`
	Roles         []*discordgo.Role `json:"roles"`
	IsAuthor      bool              `json:"isAuthor"`
	IsAdmin       bool              `json:"isAdmin"`
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

func (s *Server) getEffectiveUser(c echo.Context) *JwtCustomClaims {
	return s.hydrateClaims(GetUserFromContext(c))
}

func (s *Server) hydrateClaims(claims *JwtCustomClaims) *JwtCustomClaims {
	if claims == nil || claims.UserID == "" {
		return nil
	}

	effective := *claims
	effective.Roles = slices.Collect(slices.Values(claims.Roles))

	if userDB, err := s.db.UserByID(claims.UserID); err == nil && userDB != nil {
		effective.IsAdmin = userDB.IsAdmin
	}

	member, err := s.currentGuildMember(claims.UserID)
	if err != nil {
		log.Debug("Failed to rehydrate guild member; falling back to JWT roles", "userID", claims.UserID, "error", err)
		return &effective
	}

	effective.Roles = s.memberRoles(member, claims.Roles)
	return &effective
}

func (s *Server) currentGuildMember(userID string) (*discordgo.Member, error) {
	session := s.bot.Session()
	if session == nil {
		return nil, fmt.Errorf("discord session not ready")
	}

	member, err := session.State.Member(s.config.GuildID, userID)
	if err == nil && member != nil {
		return member, nil
	}

	member, err = session.GuildMember(s.config.GuildID, userID)
	if err != nil {
		return nil, err
	}
	return member, nil
}

func (s *Server) memberRoles(member *discordgo.Member, fallback []*discordgo.Role) []*discordgo.Role {
	if member == nil {
		return append([]*discordgo.Role(nil), fallback...)
	}

	fallbackByID := make(map[string]*discordgo.Role, len(fallback))
	for _, role := range fallback {
		if role == nil || role.ID == "" {
			continue
		}
		fallbackByID[role.ID] = role
	}

	roles := make([]*discordgo.Role, 0, len(member.Roles))
	for _, roleID := range member.Roles {
		roleID = strings.TrimSpace(roleID)
		if roleID == "" {
			continue
		}

		if role, err := s.bot.Session().State.Role(s.config.GuildID, roleID); err == nil && role != nil {
			roles = append(roles, role)
			continue
		}
		if role, ok := fallbackByID[roleID]; ok {
			roles = append(roles, role)
			continue
		}

		roles = append(roles, &discordgo.Role{ID: roleID})
	}

	return roles
}

func sessionUserFromClaims(claims *JwtCustomClaims) *SessionUser {
	if claims == nil {
		return nil
	}

	return &SessionUser{
		UserID:        claims.UserID,
		Username:      claims.Username,
		GlobalName:    claims.Username,
		Discriminator: "0000",
		Avatar:        claims.Avatar,
		Banner:        claims.Banner,
		AccentColor:   0,
		Bot:           false,
		System:        false,
		PublicFlags:   0,
		Roles:         claims.Roles,
		IsAuthor:      claims.IsAdmin,
		IsAdmin:       claims.IsAdmin,
	}
}

func (s *Server) handleSession(c echo.Context) error {
	user := s.getEffectiveUser(c)
	if user == nil || user.UserID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	return c.JSON(http.StatusOK, sessionUserFromClaims(user))
}
