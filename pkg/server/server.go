package server

import (
	"context"
	"io/fs"
	"net/http"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"drigo/app"
	"drigo/pkg/discord"
	"drigo/pkg/flight"
	"drigo/pkg/sqlite"
	"drigo/pkg/types"

	echojwt "github.com/labstack/echo-jwt/v4"
)

type GuildData struct {
	Roles    []types.CachedRole    `json:"roles"`
	Channels []types.CachedChannel `json:"channels"`
	Name     string                `json:"guild_name"`
}

type Server struct {
	router       *echo.Echo
	db           sqlite.DB
	bot          discord.Bot
	config       *Config
	preloadQueue *PreloadQueue
	getPostCache flight.Cache[sortOption, []*types.Post]
	guildCache   flight.Cache[struct{}, *GuildData]
}

type Config struct {
	Port         string
	DiscordToken string
	GuildID      string
	DB           sqlite.DB
	Bot          discord.Bot
}

func New(cfg *Config) *Server {
	if cfg.DB == nil {
		log.Fatal("DB is nil")
	}
	if cfg.Bot == nil {
		log.Fatal("Bot is nil")
	}

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
		getPostCache: flight.NewCache(func(option sortOption) ([]*types.Post, error) {
			return cfg.DB.ListPosts(option.limit, option.offset, option.sort)
		}),
		guildCache: flight.NewCache(func(_ struct{}) (*GuildData, error) {
			roles, _ := cfg.DB.GetCachedRoles()
			channels, _ := cfg.DB.GetCachedChannels()
			var guildName string
			settings, err := cfg.DB.GetSettings()
			if err == nil && settings != nil {
				// We don't have GuildName in Settings yet, but we could add it.
				// For now, if there's no name, it will be empty until fetched from Discord.
			}
			return &GuildData{
				Roles:    roles,
				Channels: channels,
				Name:     guildName,
			}, nil
		}),
	}

	s.preloadQueue = NewPreloadQueue(s)

	s.routes()

	return s
}

func (s *Server) routes() {
	s.router.GET("/posts", s.handleGetPosts)
	s.router.GET("/posts/:id", s.handleGetPost)
	s.router.GET("/roles", s.handleGetRoles)
	s.router.GET("/upload", s.handleGetUploadConfig)
	s.router.GET("/images/:id", s.handleGetImage)
	s.router.GET("/images/:id/resize", s.handleGetResizedImage)
	s.router.GET("/thumb/:id", s.handleGetThumb)
	s.router.GET("/blur/:id", s.handleGetBlur)

	s.router.GET("/login", s.handleLogin)
	s.router.GET("/auth/callback", s.handleCallback)

	s.router.POST("/posts", s.handleCreatePost)
	s.router.POST("/posts/:id/dm", s.handlePostDM)
	s.router.PATCH("/posts/:id", s.handlePatchPost)
	s.router.DELETE("/posts/:id", s.handleDeletePost)

	// Settings
	s.router.GET("/settings", s.handleGetSettings)
	s.router.POST("/settings", s.handleUpdateSettings)

	// Users
	s.router.GET("/users", s.handleGetUsers)
	s.router.POST("/users/:id/admin", s.handleToggleAdmin)

	staticFS := app.FS()
	staticFSWrapper, err := fs.Sub(staticFS, ".")
	if err != nil {
		log.Error("Failed to create sub filesystem", "error", err)
		return
	}

	fileServer := http.FileServer(http.FS(staticFSWrapper))

	s.router.GET("/*", func(c echo.Context) error {
		path := c.Request().URL.Path
		if strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/posts") ||
			strings.HasPrefix(path, "/images/") || strings.HasPrefix(path, "/thumb/") ||
			strings.HasPrefix(path, "/blur/") || strings.HasPrefix(path, "/login") ||
			strings.HasPrefix(path, "/auth/") || strings.HasPrefix(path, "/upload") ||
			strings.HasPrefix(path, "/roles") {
			return c.NoContent(http.StatusNotFound)
		}

		cleanPath := strings.TrimPrefix(path, "/")
		if cleanPath == "" {
			cleanPath = "index.html"
		}

		f, err := staticFSWrapper.Open(cleanPath)
		if err != nil {
			c.Request().URL.Path = "/"
			fileServer.ServeHTTP(c.Response(), c.Request())
			return nil
		}
		f.Close()

		fileServer.ServeHTTP(c.Response(), c.Request())
		return nil
	})
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
