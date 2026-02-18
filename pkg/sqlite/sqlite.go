package sqlite

import (
	"context"
	"database/sql"
	"errors"

	gormsqlite "gorm.io/driver/sqlite"
	"gorm.io/gorm"
	_ "modernc.org/sqlite"

	"github.com/bwmarrin/discordgo"

	"drigo/pkg/types"
)

// DB interface defines methods we need for working with allowed roles.
type DB interface {
	Stop() error
	AllowedRoles() ([]*discordgo.Role, error)
	AddRole(role *discordgo.Role) error
	IsAllowed(role *discordgo.Role) bool
	// Post operations
	CreatePost(p *types.Post) error
	ReadPost(id uint) (*types.Post, error)
	ReadPostWithBlobData(id uint) (*types.Post, error)
	ReadPostByExternalID(ext string) (*types.Post, error)
	ReadPostByExternalIDWithBlobData(ext string) (*types.Post, error)
	UpdatePost(p *types.Post) error
	DeletePost(id uint) error
	PatchPost(id uint, patch PostPatch) error
	ListPosts(limit, offset int, sort string) ([]*types.Post, error)
	UserByID(id string) (*types.User, error)
	UpsertUser(user *types.User) error
	CountUsers() (int64, error)
	SetAdmin(id string, isAdmin bool) error
	GetImageBlob(id uint) (*types.ImageBlob, error)
	GetImageThumbnailByBlobID(blobID uint) ([]byte, error)
	GetPostByBlobID(blobID uint) (*types.Post, error)
	GetSettings() (*types.Settings, error)
	UpdateSettings(settings types.Settings) (*types.Settings, error)
	GetAllUsers() ([]*types.User, error)
}

// sqliteDB is a gorm-backed implementation of DB.
type sqliteDB struct {
	db  *gorm.DB
	ctx context.Context
}

// Connect initializes the database, migrates the schema, and returns an implementation of DB.
// path can be "sqlite.db" or ":memory:" for in-memory mode.
func Connect(path string, ctx context.Context) (DB, error) {
	sqlDB, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	gdb, err := gorm.Open(
		gormsqlite.Dialector{DriverName: "sqlite", Conn: sqlDB},
		&gorm.Config{},
	)
	if err != nil {
		return nil, err
	}

	// Drop potential unique index on images.post_id to allow multiple images per post
	_ = gdb.Exec("DROP INDEX IF EXISTS idx_images_post_id").Error

	// Order matters for constraints/indexes and FKs;
	// migrate base tables first, then Post, then child tables.
	err = gdb.AutoMigrate(
		&types.User{},
		&types.Allowed{},
		&types.Post{},
		&types.Image{},
		&types.ImageBlob{},
		&types.Settings{},
	)
	if err != nil {
		return nil, err
	}
	// Ensure join table exists for many2many relation
	if err := gdb.SetupJoinTable(&types.Post{}, "AllowedRoles", &struct{}{}); err != nil {
		// Not strictly necessary; GORM will auto create. Keep for clarity.
		_ = err
	}

	return &sqliteDB{db: gdb.WithContext(ctx), ctx: ctx}, nil
}

// Stop closes the database connection.
func (s *sqliteDB) Stop() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// AllowedRoles fetches all allowed roles and returns them as []*discordgo.Role.
func (s *sqliteDB) AllowedRoles() ([]*discordgo.Role, error) {
	var allowed []types.Allowed
	if err := s.db.Find(&allowed).Error; err != nil {
		return nil, err
	}

	roles := make([]*discordgo.Role, len(allowed))
	for i, a := range allowed {
		roles[i] = &discordgo.Role{
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
	return roles, nil
}

// AddRole inserts a role into the allowed roles table if it doesn't already exist.
func (s *sqliteDB) AddRole(role *discordgo.Role) error {
	if role == nil {
		return errors.New("nil role cannot be added")
	}
	rec := types.Allowed{
		RoleID:       role.ID,
		Name:         role.Name,
		Managed:      role.Managed,
		Mentionable:  role.Mentionable,
		Hoist:        role.Hoist,
		Color:        role.Color,
		Position:     role.Position,
		Permissions:  role.Permissions,
		Icon:         role.Icon,
		UnicodeEmoji: role.UnicodeEmoji,
		Flags:        int(role.Flags),
	}
	return s.db.FirstOrCreate(&rec, types.Allowed{RoleID: role.ID}).Error
}

// IsAllowed checks if the given role exists in the allowed roles table.
func (s *sqliteDB) IsAllowed(role *discordgo.Role) bool {
	if role == nil {
		return false
	}
	var count int64
	s.db.Model(&types.Allowed{}).Where("role_id = ?", role.ID).Count(&count)
	return count > 0
}
