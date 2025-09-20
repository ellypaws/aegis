package drigo

import (
	"context"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/charmbracelet/log"

	"drigo/pkg"
	"drigo/pkg/sqlite"
)

type Bot struct {
	botSession *discordgo.Session
	mu         sync.Mutex
	context    context.Context
	db         sqlite.DB
	logger     *log.Logger
}

func (q *Bot) Stop() {
	// TODO implement me
	panic("implement me")
}

func New(botSession *discordgo.Session, ctx context.Context, db sqlite.DB, logger *log.Logger) *Bot {
	return &Bot{
		botSession: botSession,
		context:    ctx,
		db:         db,
		logger:     logger,
	}
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
