package bot

import (
	"errors"
	"fmt"

	"github.com/user/telegram-claude-bot/internal/store"
	tele "gopkg.in/telebot.v4"
)

func (b *Bot) handleFile(c tele.Context) error {
	tid := telegramID(c)
	path := normalizeCommandArg(c.Message().Payload)
	if path == "" {
		return c.Send("Usage: /file <path>")
	}

	user, _ := store.GetUser(tid)
	resolvedPath, err := b.resolveUserPath(path, userWorkingDir(user), pathTargetRegularFile)
	if err != nil {
		return c.Send(b.filePathError(err))
	}
	return b.sendFile(c, resolvedPath)
}

func (b *Bot) handleProject(c tele.Context) error {
	tid := telegramID(c)
	user, _ := store.GetUser(tid)
	dir := normalizeCommandArg(c.Message().Payload)

	if dir == "" {
		wd := "not set"
		if user != nil && user.WorkingDirectory != "" {
			wd = user.WorkingDirectory
		}
		return c.Send(fmt.Sprintf("Current working directory: %s\n\nUsage: /project <path>", wd))
	}

	resolvedDir, err := b.resolveUserPath(dir, userWorkingDir(user), pathTargetDirectory)
	if err != nil {
		return c.Send(b.projectPathError(err))
	}

	existingSession, err := store.GetSessionForDir(tid, resolvedDir)
	if err == nil && existingSession != nil {
		switchErr := store.SwitchSession(tid, existingSession.SessionID)
		if switchErr == nil {
			if err := store.SetWorkingDir(tid, resolvedDir); err != nil {
				return c.Send("Failed to set working directory.")
			}
			title := existingSession.Title
			if title == "" {
				title = existingSession.SessionID[:min(12, len(existingSession.SessionID))]
			}
			return c.Send(fmt.Sprintf("Working directory: %s\nResumed session: %s", resolvedDir, title))
		}
		if !errors.Is(switchErr, store.ErrSessionNotFound) {
			return c.Send("Failed to resume existing session.")
		}
	}

	if err := store.DeactivateSession(tid); err != nil {
		return c.Send("Failed to switch to a new session.")
	}
	if err := store.SetWorkingDir(tid, resolvedDir); err != nil {
		return c.Send("Failed to set working directory.")
	}
	return c.Send(fmt.Sprintf("Working directory: %s\nNew session will start on next message.", resolvedDir))
}
