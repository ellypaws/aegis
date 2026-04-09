package server

import (
	"context"
	"errors"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/charmbracelet/log"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"drigo/app"
	"drigo/pkg/bucket"
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
	ctx          context.Context
	cancel       context.CancelFunc
	preloadQueue *PreloadQueue
	getPostCache flight.Cache[sortOption, []*types.Post]
	guildCache   flight.Cache[struct{}, *GuildData]
	bucket       bucket.Uploader
}

type Config struct {
	Context      context.Context
	Cancel       context.CancelFunc
	Port         string
	DiscordToken string
	GuildID      string
	DB           sqlite.DB
	Bot          discord.Bot
	Bucket       bucket.Uploader
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
		ctx:    cfg.Context,
		cancel: cfg.Cancel,
		bucket: cfg.Bucket,
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

func (s *Server) Run() error {
	ctx := s.ctx
	if ctx == nil {
		ctx = context.Background()
	}

	cancel := s.cancel
	if cancel == nil {
		cancel = func() {}
	}

	signalCtx, stopSignals := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stopSignals()

	serverErrCh := make(chan error, 1)
	go func() {
		err := s.Start()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrCh <- err
			return
		}
		serverErrCh <- nil
	}()

	botErrCh := make(chan error, 1)
	go func() {
		botErrCh <- s.bot.Start()
	}()

	var runErr error
	serverDone := false
	botDone := false

	select {
	case err := <-serverErrCh:
		serverDone = true
		if err != nil {
			log.Debug("HTTP server exited with error", "error", err)
			runErr = err
		} else {
			log.Debug("HTTP server exited cleanly")
		}
	case err := <-botErrCh:
		botDone = true
		if err != nil {
			log.Debug("Discord bot exited with error", "error", err)
			runErr = err
		} else {
			log.Debug("Discord bot exited cleanly")
		}
	case <-signalCtx.Done():
		log.Debug("Shutdown signal received", "error", signalCtx.Err())
	}

	log.Info("Gracefully shutting down.")

	cancel()

	if !serverDone {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		if err := s.Shutdown(shutdownCtx); err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, http.ErrServerClosed) {
			log.Debug("HTTP server shutdown returned error", "error", err)
			runErr = errors.Join(runErr, err)
		} else {
			log.Debug("HTTP server shutdown request completed")
		}
		shutdownCancel()

		select {
		case err := <-serverErrCh:
			serverDone = true
			if err != nil {
				log.Debug("HTTP server exited after shutdown with error", "error", err)
				runErr = errors.Join(runErr, err)
			} else {
				log.Debug("HTTP server exited after shutdown")
			}
		case <-time.After(10 * time.Second):
			log.Debug("Timed out waiting for HTTP server to exit after shutdown")
		}
	}

	if !botDone {
		select {
		case err := <-botErrCh:
			botDone = true
			if err != nil {
				log.Debug("Discord bot exited after shutdown with error", "error", err)
				runErr = errors.Join(runErr, err)
			} else {
				log.Debug("Discord bot exited after shutdown")
			}
		case <-time.After(10 * time.Second):
			log.Debug("Timed out waiting for Discord bot to exit after shutdown")
		}
	}

	if err := s.db.Stop(); err != nil {
		log.Debug("SQLite shutdown returned error", "error", err)
		runErr = errors.Join(runErr, err)
	} else {
		log.Debug("SQLite shutdown completed")
	}

	return runErr
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
	s.router.GET("/auth/session", s.handleSession)

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
