package utils

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/lucasb-eyer/go-colorful"

	"github.com/bwmarrin/discordgo"
)

type HasUser interface {
	*discordgo.User |
		*discordgo.Member |
		*discordgo.Message |
		*discordgo.MessageCreate |
		*discordgo.MessageUpdate |
		*discordgo.MessageDelete |
		*discordgo.Interaction |
		*discordgo.InteractionCreate |
		*discordgo.MessageInteraction |
		*discordgo.MessageInteractionMetadata
}

func GetUsername[T HasUser](entities ...T) string {
	if user := GetUser(ToAny(entities)); user != nil {
		return user.Username
	}
	return "unknown"
}

func GetDisplayName[T HasUser](entities ...T) string {
	if user := GetUser(ToAny(entities)); user != nil {
		return user.DisplayName()
	}
	return "unknown"
}

func ToAny[S ~[]T, T any](a S) []any {
	if a == nil {
		return nil
	}
	ret := make([]any, len(a))
	for i, v := range a {
		ret[i] = v
	}
	return ret
}

func GetUser(entities ...any) *discordgo.User {
	for _, entity := range entities {
		v := reflect.ValueOf(entity)
		if v.Kind() == reflect.Pointer && v.IsNil() {
			continue
		}
		switch e := entity.(type) {
		case *discordgo.User:
			return e
		case *discordgo.Member:
			return GetUser(e.User)
		case *discordgo.Message:
			return GetUser(e.Author, e.Member)
		case *discordgo.MessageCreate:
			return GetUser(e.Message)
		case *discordgo.MessageUpdate:
			return GetUser(e.Message, e.BeforeUpdate)
		case *discordgo.MessageDelete:
			return GetUser(e.Message, e.BeforeDelete)
		case *discordgo.Interaction:
			return GetUser(e.Member, e.User)
		case *discordgo.InteractionCreate:
			return GetUser(e.Interaction)
		case *discordgo.MessageInteraction:
			return GetUser(e.User, e.Member)
		case *discordgo.MessageInteractionMetadata:
			return GetUser(e.User)
		default:
			continue
		}
	}
	return nil
}

func GetReference(s *discordgo.Session, r *discordgo.MessageReference) (*discordgo.Message, error) {
	if s == nil {
		return nil, errors.New("*discordgo.Session is nil")
	}
	if r == nil {
		return nil, errors.New("*discordgo.MessageReference is nil")
	}
	retrieve, err := s.State.Message(r.ChannelID, r.MessageID)
	if err != nil {
		if errors.Is(err, discordgo.ErrStateNotFound) {
			retrieve, err = s.ChannelMessage(r.ChannelID, r.MessageID)
			if err != nil {
				return nil, err
			}
			err = s.State.MessageAdd(retrieve)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	return retrieve, nil
}

var invalidChars = regexp.MustCompile(`[^-_'\p{L}\p{N}]`)

// DetectInvalidChars returns an error showing positions of invalid characters.
// Example:
//
//	input: "He!!o"
//	output:
//	  He!!o
//	    ^^
func DetectInvalidChars(name string) error {
	locs := invalidChars.FindAllStringIndex(name, -1)
	if len(locs) == 0 {
		return nil
	}

	col := colorful.Hsv(4, 0.67, 1.0)
	r, g, b := col.RGB255()
	ansiStart := fmt.Sprintf("\033[38;2;%d;%d;%dm", r, g, b)
	ansiReset := "\033[0m"

	highlighted := []rune(name)
	coloredOutput := ""
	for i := 0; i < len(highlighted); i++ {
		isInvalid := false
		for _, loc := range locs {
			if i >= loc[0] && i < loc[1] {
				isInvalid = true
				break
			}
		}
		if isInvalid {
			coloredOutput += ansiStart + string(highlighted[i]) + ansiReset
		} else {
			coloredOutput += string(highlighted[i])
		}
	}

	marker := make([]rune, len(highlighted))
	for i := range marker {
		marker[i] = ' '
	}
	for _, loc := range locs {
		for i := loc[0]; i < loc[1]; i++ {
			marker[i] = '^'
		}
	}

	msg := fmt.Sprintf("%s\n%s", coloredOutput, string(marker))
	return fmt.Errorf("invalid characters detected:\n%s", msg)
}

// StripInvalidName removes invalid characters and normalizes casing.
// Rules:
//   - Allowed: letters, numbers, dash, underscore, apostrophe
//   - If the name has *any* lowercase, force the entire result to lowercase
//   - If the name is entirely uppercase, leave it uppercase
func StripInvalidName(name string) string {
	// Strip invalid characters
	clean := invalidChars.ReplaceAllString(name, "")

	if clean == "" {
		return clean
	}

	lower := strings.ToLower(clean)
	if lower != clean {
		if strings.ToUpper(clean) == clean {
			return clean
		}
		return lower
	}

	return clean
}

type AttachmentImage struct {
	Attachment *discordgo.MessageAttachment
	Image      *Image
}

func GetAttachments(i *discordgo.InteractionCreate) (map[string]AttachmentImage, error) {
	if i.ApplicationCommandData().Resolved == nil {
		return nil, nil
	}

	resolved := i.ApplicationCommandData().Resolved.Attachments
	if resolved == nil {
		return nil, nil
	}

	attachments := make(map[string]AttachmentImage, len(resolved))
	for snowflake, attachment := range resolved {
		log.Printf("Attachment[%v]: %#v", snowflake, attachment.URL)
		if !strings.HasPrefix(attachment.ContentType, "image/") && !strings.HasPrefix(attachment.ContentType, "video/") {
			log.Printf("Attachment[%v] is not an image or video, removing from queue.", snowflake)
			continue
		}

		attachments[snowflake] = AttachmentImage{attachment, AsyncImage(attachment.URL)}
	}

	return attachments, nil
}

func GetOpts(data discordgo.ApplicationCommandInteractionData) map[string]*discordgo.ApplicationCommandInteractionDataOption {
	options := data.Options
	optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
	for _, opt := range options {
		optionMap[opt.Name] = opt
	}
	return optionMap
}
