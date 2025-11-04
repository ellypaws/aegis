# Aegis

## Summary

Aegis is a versatile Discord bot developed in Go, utilizing the powerful `discordgo` library for seamless interaction with the Discord API. It features a lightweight SQLite database for data persistence and offers optional integration with Amazon S3 for file storage. The bot is designed to be easily configurable through environment variables or command-line flags, making it adaptable to various deployment environments.

The main purpose of this bot is to allow content creators to share exclusive images with specific roles in a Discord server.

## Core Functionality

Aegis's primary feature is the ability to create posts with a hidden full-resolution image, accessible only to users with specific roles. When a new post is created, the bot displays a message with a low-resolution thumbnail. Users with the designated roles can then click a "Show me this image" button to reveal the full-resolution image in an ephemeral message, visible only to them. They can also choose to have the image sent to their DMs.

This functionality is ideal for content creators, artists, or any community that wants to share exclusive content with specific member tiers or roles.

## Usage

### Configuration

To get started with Aegis, you can configure the bot using either a `.env` file or command-line flags.

> [!IMPORTANT]
> A `BOT_TOKEN` is required to run the bot. You can get this from the Discord Developer Portal.

```env
BOT_TOKEN="YOUR_BOT_TOKEN_HERE"
GUILD_ID="YOUR_GUILD_ID_HERE"
REMOVE_COMMANDS=false

# Optional S3 Configuration
ACCESS_KEY="YOUR_S3_ACCESS_KEY"
SECRET_KEY="YOUR_S3_SECRET_KEY"
S3_BUCKET="YOUR_S3_BUCKET_NAME"
S3_REGION="YOUR_S3_REGION"
S3_ENDPOINT="YOUR_S3_ENDPOINT"
S3_PREFIX="YOUR_S3_PREFIX"
```

Alternatively, you can use the following command-line flags when running the bot:

```bash
go run ./cmd -token="YOUR_BOT_TOKEN" -guild="YOUR_GUILD_ID" -remove=false
```

> [!WARNING]
> Be careful with the `-remove=true` flag or `REMOVE_COMMANDS=true` environment variable, as it will delete all of your bot's slash commands from your server when the bot shuts down.

### Commands

Once the bot is running, you can use the following slash commands in your Discord server:

- `/post`: Creates a new image post. This command has the following options:
    - `full`: The full-resolution image.
    - `thumbnail`: The thumbnail image to be displayed initially.
    - `title`: An optional title for the post.
    - `description`: An optional description for the post.
    - `allowed_roles`: The role that is allowed to view the full image.
    - `allowed_channels`: The channel where the post will be sent.

> [!TIP]
> If you don't specify `allowed_roles` or `allowed_channels` when using the `/post` command, a modal will pop up allowing you to select multiple roles and channels. This is useful for crossposting to several channels at once or granting access to multiple roles.

When a user with an allowed role clicks the "Show me this image" button, the bot will display the full image in a message only visible to that user. The "Send to DMs" button will send the full image to the user's direct messages.
