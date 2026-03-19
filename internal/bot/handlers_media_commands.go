package bot

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/TrungyuD/telegram-chat-resume-bot/internal/claude"
	tele "gopkg.in/telebot.v4"
)

const maxPhotoSize = 10 * 1024 * 1024  // 10MB
const maxDocSize = 20 * 1024 * 1024    // 20MB

func (b *Bot) handleText(c tele.Context) error {
	return b.sendToClaude(c, c.Text(), "full")
}

func (b *Bot) handlePhoto(c tele.Context) error {
	photo := c.Message().Photo
	if photo == nil {
		return c.Send("No photo found.")
	}

	reader, err := b.tele.File(&photo.File)
	if err != nil {
		return c.Send("Failed to download photo.")
	}
	defer reader.Close()

	data, err := io.ReadAll(io.LimitReader(reader, maxPhotoSize+1))
	if err != nil {
		return c.Send("Failed to read photo.")
	}
	if len(data) > maxPhotoSize {
		return c.Send("Photo too large (max 10MB).")
	}

	b64 := base64.StdEncoding.EncodeToString(data)
	mediaType := "image/jpeg"
	if strings.HasSuffix(photo.FilePath, ".png") {
		mediaType = "image/png"
	}

	caption := c.Message().Caption
	if caption == "" {
		caption = "What's in this image?"
	}

	return b.sendToClaudeWithImages(c, caption, []claude.ImageInput{{Base64: b64, MediaType: mediaType}}, "full")
}

func (b *Bot) handleDocument(c tele.Context) error {
	doc := c.Message().Document
	if doc == nil {
		return c.Send("No document found.")
	}
	if doc.FileSize > maxDocSize {
		return c.Send("File too large (max 20MB).")
	}

	reader, err := b.tele.File(&doc.File)
	if err != nil {
		return c.Send("Failed to download document.")
	}
	defer reader.Close()

	data, err := io.ReadAll(io.LimitReader(reader, maxDocSize+1))
	if err != nil {
		return c.Send("Failed to read document.")
	}
	if len(data) > maxDocSize {
		return c.Send("File too large (max 20MB).")
	}

	contentType := http.DetectContentType(data)
	if strings.HasPrefix(contentType, "image/") {
		b64 := base64.StdEncoding.EncodeToString(data)
		caption := c.Message().Caption
		if caption == "" {
			caption = "What's in this image?"
		}
		return b.sendToClaudeWithImages(c, caption, []claude.ImageInput{{Base64: b64, MediaType: contentType}}, "full")
	}

	text := string(data)
	if len(text) > 50000 {
		text = text[:50000] + "\n... (truncated)"
	}

	message := fmt.Sprintf("File: %s\n\n```\n%s\n```", doc.FileName, text)
	if c.Message().Caption != "" {
		message = c.Message().Caption + "\n\n" + message
	}

	return b.sendToClaude(c, message, "full")
}
