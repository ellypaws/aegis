package discord

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"os/signal"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/log"

	"github.com/bwmarrin/discordgo"

	"drigo/pkg"
	"drigo/pkg/bucket"
	"drigo/pkg/discord/handlers"
	"drigo/pkg/drigo"
	"drigo/pkg/sqlite"
	"drigo/pkg/utils"
)

type botImpl struct {
	session *discordgo.Session

	registeredCommands map[string]*discordgo.ApplicationCommand
	config             *Config

	queues []pkg.HandlerStartStopper

	handlers   pkg.CommandHandlers
	components pkg.Components

	ready chan struct{}
}

type Config struct {
	Context        context.Context
	BotToken       string
	GuildID        string
	DrigoBot       pkg.Queue
	Database       sqlite.DB
	RemoveCommands bool
	Bucket         bucket.Uploader
}

type Bot interface {
	Start() error
	Session() *discordgo.Session
	Ready() <-chan struct{}
}

func New(cfg *Config) (Bot, error) {
	if cfg.BotToken == "" {
		return nil, errors.New("missing bot token")
	}

	handlers.Token = &cfg.BotToken

	if cfg.GuildID == "" {
		// return nil, errors.New("missing guild ID")
		log.Printf("Guild ID not provided, commands will be registered globally")
	}

	botSession, err := discordgo.New("Bot " + cfg.BotToken)
	if err != nil {
		return nil, err
	}

	cfg.DrigoBot = drigo.New(botSession, cfg.Context, cfg.Database, log.Default(), cfg.Bucket)
	queues := []pkg.HandlerStartStopper{
		cfg.DrigoBot,
	}
	queues = slices.DeleteFunc(queues, func(q pkg.HandlerStartStopper) bool { return q == nil })

	bot := &botImpl{
		session:            botSession,
		registeredCommands: make(map[handlers.Command]*discordgo.ApplicationCommand),
		config:             cfg,
		queues:             queues,
		handlers:           make(pkg.CommandHandlers),
		components:         handlers.ComponentHandlers,
		ready:              make(chan struct{}),
	}

	return bot, nil
}

func (b *botImpl) Session() *discordgo.Session {
	return b.session
}

func (b *botImpl) Ready() <-chan struct{} {
	if b.ready == nil {
		closed := make(chan struct{})
		close(closed)
		return closed
	}
	return b.ready
}

func (b *botImpl) registerHandlers() {
	for _, q := range b.queues {
		handlers := q.Handlers()
		for interactionType, commandHandlers := range handlers {
			if _, ok := b.handlers[interactionType]; !ok {
				maps.Copy(b.handlers, handlers)
			} else {
				maps.Copy(b.handlers[interactionType], commandHandlers)
			}
		}

		maps.Copy(b.components, q.Components())
	}

	b.session.AddHandler(func(session *discordgo.Session, i *discordgo.InteractionCreate) {
		var handler pkg.Handler
		var ok bool

		handleName := getHandleName(i)
		if i.Type == discordgo.InteractionMessageComponent {
			log.Printf("Component with customID `%v` was pressed, attempting to respond\n", i.MessageComponentData().CustomID)
			// Allow dynamic custom IDs (e.g., "show_image:<postKey>") by prefix matching
			for name, h := range b.components {
				if strings.HasPrefix(handleName, name) {
					handler = h
					ok = true
					break
				}
			}
		} else {
			handles, exist := b.handlers[i.Type]
			if !exist {
				log.Printf("Unknown interaction type: %v", i.Type)
				return
			}

			for name, h := range handles {
				if strings.HasPrefix(handleName, name) {
					handler = h
					ok = true
					break
				}
			}
		}

		if !ok || handler == nil {
			var interactionType = "unknown"
			var interactionName = "unknown"
			switch i.Type {
			case discordgo.InteractionApplicationCommand:
				interactionType = "command"
				interactionName = handleName
			case discordgo.InteractionMessageComponent:
				interactionType = "component"
				interactionName = handleName
			case discordgo.InteractionApplicationCommandAutocomplete:
				interactionType = "autocomplete"

				data := i.ApplicationCommandData()
				for _, opt := range data.Options {
					if !opt.Focused {
						continue
					}
					interactionName = fmt.Sprintf("command: /%s option: %s", data.Name, opt.Name)
					break
				}
			case discordgo.InteractionModalSubmit:
				interactionType = "modal"
				interactionName = handleName
			}
			log.Printf("WARNING: Cannot find handler for interaction [%v] '%v'", interactionType, interactionName)
			return
		}

		err := handler(session, i)

		if err != nil {
			username := utils.GetUsername(i.Interaction)
			if errors.Is(err, handlers.ResponseError) {
				log.Printf("Error responding to interaction for %s: %v", username, err)
				return
			}
			err := handlers.ErrorEdit(session, i.Interaction, err)
			if err != nil {
				log.Printf("Error showing error message to user %s: %v", username, err)
			}
		}
	})
}

func getHandleName(i *discordgo.InteractionCreate) string {
	switch i.Type {
	case discordgo.InteractionApplicationCommand, discordgo.InteractionApplicationCommandAutocomplete:
		return i.ApplicationCommandData().Name
	case discordgo.InteractionMessageComponent:
		return i.MessageComponentData().CustomID
	case discordgo.InteractionModalSubmit:
		return i.ModalSubmitData().CustomID
	default:
		return "unknown"
	}
}

func (b *botImpl) registerCommands() error {
	b.registeredCommands = make(map[handlers.Command]*discordgo.ApplicationCommand)

	for _, q := range b.queues {
		if q == nil {
			continue
		}

		for _, command := range q.Commands() {
			cmd, err := b.session.ApplicationCommandCreate(b.session.State.User.ID, b.config.GuildID, command)
			if err != nil {
				return fmt.Errorf("cannot create '%s' command: %w", command.Name, err)
			}

			b.registeredCommands[command.Name] = cmd
			log.Printf("Registered %v command as: /%v", command.Name, cmd.Name)
		}
	}

	return nil
}

func (b *botImpl) Start() error {
	b.session.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
		if b.ready != nil {
			close(b.ready)
		} else {
			log.Print("Warning: ready channel is nil")
		}
	})

	err := b.session.Open()
	if err != nil {
		return fmt.Errorf("error opening connection to Discord: %w", err)
	}

	err = b.registerCommands()
	if err != nil {
		return fmt.Errorf("error registering commands: %w", err)
	}

	b.registerHandlers()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	queues := []pkg.StartStop{
		b.config.DrigoBot,
	}

	queues = slices.DeleteFunc(queues, isNil)
	for _, q := range queues {
		go q.Start(b.session)
	}

	if len(queues) == 0 {
		log.Error("No queues to start, exiting...")
		stop()
	} else {
		log.Info("Press Ctrl+C to exit")
	}

	<-ctx.Done()
	var wg sync.WaitGroup
	for _, q := range queues {
		wg.Go(stopQueue(q))
	}
	wg.Wait()

	err = b.teardown()
	if err != nil {
		return fmt.Errorf("error tearing down bot: %w", err)
	}

	return nil
}

func isNil(q pkg.StartStop) bool {
	return q == nil
}

func stopQueue(q pkg.StartStop) func() {
	return func() {
		stopped := make(chan struct{})
		go func() {
			q.Stop()
			close(stopped)
		}()
		select {
		case <-stopped:
			return
		case <-time.After(10 * time.Second):
			panic("queue did not stop in time")
		}
	}
}

func (b *botImpl) teardown() error {
	if b.config.RemoveCommands {
		log.Printf("Removing all commands added by bot...")

		for key, v := range b.registeredCommands {
			log.Printf("Removing command [key:%v], '%v'...", key, v.Name)

			err := b.session.ApplicationCommandDelete(b.session.State.User.ID, b.config.GuildID, v.ID)
			if err != nil {
				log.Errorf("Cannot delete '%v' command: %v", v.Name, err)
				panic(err)
			}
		}
	}

	return b.session.Close()
}
