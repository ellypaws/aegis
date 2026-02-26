package main

import (
	"cmp"
	"context"
	"flag"
	"os"
	"os/signal"

	"github.com/charmbracelet/log"
	"github.com/joho/godotenv"

	"drigo/pkg/bucket"
	"drigo/pkg/discord"
	"drigo/pkg/server"
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

	// Optional S3 configuration
	accessKey := os.Getenv("ACCESS_KEY")
	secretKey := os.Getenv("SECRET_KEY")
	s3Bucket := os.Getenv("S3_BUCKET")
	s3Region := os.Getenv("S3_REGION")
	s3Endpoint := os.Getenv("S3_ENDPOINT")
	s3Prefix := os.Getenv("S3_PREFIX")

	var uploader bucket.Uploader
	if accessKey != "" && secretKey != "" && s3Bucket != "" {
		// Construct S3 uploader; errors are non-fatal, we'll log and continue without fallback uploads
		if s3Region == "" {
			s3Region = "us-east-1"
		}
		s3u, err := bucket.NewS3(ctx, accessKey, secretKey, s3Region, s3Bucket, s3Endpoint, s3Prefix)
		if err != nil {
			log.Errorf("Failed to initialize S3 uploader: %v", err)
		} else {
			uploader = s3u
		}
	}

	bot, err := discord.New(&discord.Config{
		Context:        ctx,
		BotToken:       *botToken,
		GuildID:        *guildID,
		Database:       sqliteDB,
		RemoveCommands: removeCommands,
		Bucket:         uploader,
	})
	if err != nil {
		log.Fatalf("Error creating Discord bot: %v", err)
	}

	srv := server.New(&server.Config{
		Port:         cmp.Or(os.Getenv("PORT"), "3000"),
		DiscordToken: *botToken,
		GuildID:      *guildID,
		DB:           sqliteDB,
		Bot:          bot,
		Bucket:       uploader,
	})

	go func() {
		if err := srv.Start(); err != nil {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	if err := bot.Start(); err != nil {
		panic(err)
	}

	log.Info("Gracefully shutting down.")
}
