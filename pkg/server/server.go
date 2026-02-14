package server

import (
	"context"

	"github.com/charmbracelet/log"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"drigo/pkg/discord"
	"drigo/pkg/sqlite"

	echojwt "github.com/labstack/echo-jwt/v4"
)

type Server struct {
	router *echo.Echo
	db     sqlite.DB
	bot    discord.Bot
	config *Config
}

type Config struct {
	Port         string
	DiscordToken string
	GuildID      string
	DB           sqlite.DB
	Bot          discord.Bot
}

func New(cfg *Config) *Server {
	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{echo.GET, echo.POST, echo.PUT, echo.DELETE, echo.OPTIONS, echo.PATCH},
		AllowHeaders:     []string{echo.HeaderContentType, echo.HeaderAuthorization, "X-CSRF-Token"},
		ExposeHeaders:    []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	jwtConfig := echojwt.Config{
		NewClaimsFunc: func(c echo.Context) jwt.Claims {
			return new(JwtCustomClaims)
		},
		SigningKey:  GetJWTSecret(),
		TokenLookup: "header:Authorization:Bearer ,query:token",
		ErrorHandler: func(c echo.Context, err error) error {
			if err != nil {
				log.Debug("JWT middleware error (token missing or invalid, continuing)", "path", c.Path(), "error", err)
			}
			return nil
		},
		SuccessHandler: func(c echo.Context) {
			log.Debug("JWT token parsed successfully", "path", c.Path())
		},
		ContinueOnIgnoredError: true,
	}
	e.Use(echojwt.WithConfig(jwtConfig))

	s := &Server{
		router: e,
		db:     cfg.DB,
		bot:    cfg.Bot,
		config: cfg,
	}

	s.routes()

	return s
}

func (s *Server) routes() {
	s.router.GET("/posts", s.handleGetPosts)
	s.router.GET("/posts/:id", s.handleGetPost) // Get by post-key
	s.router.GET("/roles", s.handleGetRoles)
	s.router.GET("/upload", s.handleGetUploadConfig)

	s.router.GET("/login", s.handleLogin)
	s.router.GET("/auth/callback", s.handleCallback)

	s.router.POST("/posts", s.handleCreatePost)
	s.router.PATCH("/posts/:id", s.handlePatchPost)
}

func (s *Server) Start() error {
	port := s.config.Port
	if port == "" {
		port = "3000"
	}

	return s.router.Start(":" + port)
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.router.Shutdown(ctx)
}
