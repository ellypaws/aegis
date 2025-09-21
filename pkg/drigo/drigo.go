package drigo

import (
	"context"
	"sync"
	"time"

	"github.com/charmbracelet/log"

	"github.com/bwmarrin/discordgo"

	"drigo/pkg"
	"drigo/pkg/sqlite"
)

type Bot struct {
	botSession *discordgo.Session
	mu         sync.Mutex
	context    context.Context
	db         sqlite.DB
	logger     *log.Logger
	pending    map[string]*PendingPost // ksuid -> pending post info until modal submit
	msgToPost  map[string]string       // messageID -> postKey
}

func (q *Bot) Stop() {
	q.logger.Info("Stopping drigo bot...")
	q.mu.Lock()
	defer q.mu.Unlock()
	for k := range q.pending {
		delete(q.pending, k)
	}
	for k := range q.msgToPost {
		delete(q.msgToPost, k)
	}
	q.logger.Info("Drigo bot stopped.")
}

func New(botSession *discordgo.Session, ctx context.Context, db sqlite.DB, logger *log.Logger) *Bot {
	return &Bot{
		botSession: botSession,
		context:    ctx,
		db:         db,
		logger:     logger,
		pending:    make(map[string]*PendingPost),
		msgToPost:  make(map[string]string),
	}
}

// PendingPost holds data collected from the slash command until the user completes the modal.
type PendingPost struct {
	PostKey     string
	GuildID     string
	ChannelID   string
	Author      *discordgo.User
	Thumbnail   []byte
	Full        []byte
	Title       string
	Description string
}

func (q *Bot) Commands() []*discordgo.ApplicationCommand { return q.commands() }
func (q *Bot) Handlers() pkg.CommandHandlers             { return q.handlers() }
func (q *Bot) Components() pkg.Components                { return q.components() }

func (q *Bot) Start(botSession *discordgo.Session) {
	q.botSession = botSession

Polling:
	for {
		select {
		case <-q.context.Done():
			break Polling
		case <-time.After(1 * time.Second):

		}
	}
}
