package handlers

import (
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"time"

	"drigo/pkg/types"
	"drigo/pkg/utils"
	"github.com/bwmarrin/discordgo"

	"drigo/pkg/compositor"
)

// EmbedImages modifies the provided webhook to include the provided embed and images.
// If there are more than four images, they will be tiled into a single image.
// images and thumbnails are expected to be in bytes and not base64 encoded.
func EmbedImages(webhook *discordgo.WebhookEdit, embed *discordgo.MessageEmbed, images, thumbnails []io.Reader, compositor compositor.Renderer) error {
	if webhook == nil {
		return errors.New("imageEmbedFromBuffers called with nil webhook")
	}
	now := time.Now().UTC()
	nowFormatted := now.Format("2006-01-02_15-04-05")
	if embed == nil {
		embed = &discordgo.MessageEmbed{
			Type:        discordgo.EmbedTypeImage,
			Timestamp:   now.Format(time.RFC3339),
			Description: "Title",
		}
	}

	var files []*discordgo.File

	embedCount := len(images)
	if embedCount > 4 {
		embedCount = 1
	}
	embeds := make([]*discordgo.MessageEmbed, 1, embedCount+1)
	embeds[0] = embed

	thumbnails = slices.DeleteFunc(thumbnails, func(i io.Reader) bool { return i == nil })
	if len(thumbnails) > 0 {
		thumbnailTile, err := compositor.TileImages(thumbnails)
		if err != nil {
			return fmt.Errorf("error tiling thumbnails: %w", err)
		}

		if thumbnailTile != nil {
			embed.Thumbnail = &discordgo.MessageEmbedThumbnail{
				URL: "attachment://thumbnail.png",
			}
			files = append(files, &discordgo.File{
				Name:   "thumbnail.png",
				Reader: thumbnailTile,
			})
		}
	}

	images = slices.DeleteFunc(images, func(i io.Reader) bool { return i == nil })
	if len(images) > 4 { // Tile images if more than four
		if compositor == nil {
			return errors.New("compositor is required for tiling more than four images")
		}

		primaryTile, err := compositor.TileImages(images)
		if err != nil {
			return fmt.Errorf("error tiling primary images: %w", err)
		}

		// Use primaryTile as the only image
		if primaryTile != nil {
			images[0] = primaryTile
			clear(images[1:])
			images = images[:1]
		}
	}

	// Create separate embeds for four or fewer images
	for i, imgBuf := range images {
		if imgBuf == nil {
			continue
		}

		contentType := "image/png"
		extension := "png"
		if contentTypeInterface, ok := imgBuf.(types.ReaderType); ok {
			contentType = contentTypeInterface.ContentType()
			extension = utils.GetFileExtension(contentType)
		}

		imgName := fmt.Sprintf("%v-%d.%s", nowFormatted, i, extension)
		files = append(files, &discordgo.File{
			Name:        imgName,
			ContentType: contentType,
			Reader:      imgBuf,
		})

		var embedType discordgo.EmbedType
		switch {
		case strings.HasSuffix(contentType, "gif"):
			embedType = discordgo.EmbedTypeGifv
		case strings.HasPrefix(contentType, "image/"):
			embedType = discordgo.EmbedTypeImage
		case strings.HasPrefix(contentType, "video/"):
			embedType = discordgo.EmbedTypeVideo
			continue
		default:
			embedType = discordgo.EmbedTypeImage
		}

		embeds = append(embeds, &discordgo.MessageEmbed{
			Type: embedType,
			URL:  "https://github.com/ellypaws",
			Image: &discordgo.MessageEmbedImage{
				URL: fmt.Sprintf("attachment://%s", imgName),
			},
		})
	}

	webhook.Embeds = &embeds
	webhook.Files = files
	return nil
}
