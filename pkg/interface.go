package pkg

import (
	"github.com/bwmarrin/discordgo"
)

type Queue interface {
	Start(botSession *discordgo.Session)

	Registrar

	Stop()

	SendDirectMessage(userID, postID string) error
}

type Handler = func(*discordgo.Session, *discordgo.InteractionCreate) error

type Command = string
type CommandHandlers = map[discordgo.InteractionType]map[Command]Handler
type Components = map[string]Handler

type Registrar interface {
	Commands() []*discordgo.ApplicationCommand
	Handlers() CommandHandlers
	Components() Components
}

type HandlerStartStopper interface {
	Registrar
	StartStop
}

type StartStop interface {
	Start(botSession *discordgo.Session)
	Stop()
}
