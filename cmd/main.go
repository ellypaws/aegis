package main

import (
	"context"
	"flag"
	"os"
	"os/signal"

	"github.com/charmbracelet/log"
	"github.com/joho/godotenv"

	"drigo/pkg/discord"
	"drigo/pkg/sqlite"
)

// Bot parameters
var (
	guildID            = flag.String("guild", "", "Guild ID. If not passed - bot registers commands globally")
	botToken           = flag.String("token", "", "Bot access token")
	removeCommandsFlag = flag.Bool("remove", false, "Delete all commands when bot exits")
)

func init() {
	log.Default().SetLevel(log.DebugLevel)
	if err := godotenv.Load(); err != nil {
		log.Error("Error loading .env file", "error", err)
	}
	log.Info(".env file loaded successfully")

	if botToken == nil || *botToken == "" {
		tokenEnv := os.Getenv("BOT_TOKEN")
		if tokenEnv == "YOUR_BOT_TOKEN_HERE" {
			log.Fatalf("Invalid bot token from .env file: %v\n"+
				"Did you edit the .env or run the program with -token ?", tokenEnv)
		}
		if tokenEnv != "" {
			botToken = &tokenEnv
		}
	}

	if guildID == nil || *guildID == "" {
		guildEnv := os.Getenv("GUILD_ID")
		if guildEnv != "" {
			guildID = &guildEnv
		}
	}

	if removeCommandsFlag == nil || !*removeCommandsFlag {
		removeCommandsEnv := os.Getenv("REMOVE_COMMANDS")
		if removeCommandsEnv != "" {
			removeCommandsFlag = new(bool)
			*removeCommandsFlag = removeCommandsEnv == "true"
		}
	}
}

func main() {
	flag.Parse()

	if botToken == nil || *botToken == "" {
		log.Fatalf("Bot token flag is required")
	}

	var removeCommands bool

	if removeCommandsFlag != nil && *removeCommandsFlag {
		removeCommands = *removeCommandsFlag
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	sqliteDB, err := sqlite.Connect("sqlite.db", ctx)
	if err != nil {
		log.Fatalf("Failed to create sqlite database: %v", err)
	}

	bot, err := discord.New(&discord.Config{
		Context:        ctx,
		BotToken:       *botToken,
		GuildID:        *guildID,
		Database:       sqliteDB,
		RemoveCommands: removeCommands,
	})
	if err != nil {
		log.Fatalf("Error creating Discord bot: %v", err)
	}

	if err := bot.Start(); err != nil {
		panic(err)
	}

	log.Info("Gracefully shutting down.")
}
