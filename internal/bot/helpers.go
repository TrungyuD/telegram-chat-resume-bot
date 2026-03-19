package bot

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/user/telegram-claude-bot/internal/format"
	tele "gopkg.in/telebot.v4"
)

var (
	errPathNotAllowed      = errors.New("path not allowed")
	errPathNotFound        = errors.New("path not found")
	errExpectedDirectory   = errors.New("path is not a directory")
	errExpectedRegularFile = errors.New("path is not a regular file")
)

type pathTargetKind int

const (
	pathTargetDirectory pathTargetKind = iota
	pathTargetRegularFile
)

// sendLong splits a long message and sends chunks.
func (b *Bot) sendLong(c tele.Context, text string, parseMode tele.ParseMode) error {
	chunks := format.SplitMessage(text, 4096)
	for i, chunk := range chunks {
		opts := &tele.SendOptions{}
		if parseMode != "" {
			opts.ParseMode = parseMode
		}
		if i == 0 {
			if _, err := b.tele.Send(c.Chat(), chunk, opts); err != nil {
				if parseMode != "" {
					if _, err2 := b.tele.Send(c.Chat(), chunk); err2 != nil {
						return err2
					}
				} else {
					return err
				}
			}
		} else {
			if _, err := b.tele.Send(c.Chat(), chunk, opts); err != nil {
				return err
			}
		}
	}
	return nil
}

func (b *Bot) resolveUserPath(rawPath, baseDir string, kind pathTargetKind) (string, error) {
	candidate := strings.TrimSpace(rawPath)
	if candidate == "" {
		return "", errPathNotFound
	}
	if !filepath.IsAbs(candidate) {
		candidate = filepath.Join(baseDir, candidate)
	}

	absPath, err := filepath.Abs(candidate)
	if err != nil {
		return "", fmt.Errorf("resolve path: %w", err)
	}

	if _, err := os.Stat(absPath); err != nil {
		if os.IsNotExist(err) {
			return "", errPathNotFound
		}
		return "", fmt.Errorf("stat path: %w", err)
	}

	canonicalPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		return "", fmt.Errorf("resolve symlinks: %w", err)
	}
	canonicalPath = filepath.Clean(canonicalPath)

	info, err := os.Stat(canonicalPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", errPathNotFound
		}
		return "", fmt.Errorf("stat canonical path: %w", err)
	}

	switch kind {
	case pathTargetDirectory:
		if !info.IsDir() {
			return "", errExpectedDirectory
		}
	case pathTargetRegularFile:
		if !info.Mode().IsRegular() {
			return "", errExpectedRegularFile
		}
	}

	if !b.isAllowedPath(canonicalPath) {
		return "", errPathNotAllowed
	}

	return canonicalPath, nil
}

func (b *Bot) isAllowedPath(canonicalPath string) bool {
	allowed := b.config.AllowedWorkingDirs
	if len(allowed) == 0 {
		return true
	}
	for _, root := range allowed {
		canonicalRoot, err := canonicalizeExistingPath(root)
		if err != nil {
			continue
		}
		if pathWithinRoot(canonicalPath, canonicalRoot) {
			return true
		}
	}
	return false
}

func canonicalizeExistingPath(path string) (string, error) {
	absPath, err := filepath.Abs(strings.TrimSpace(path))
	if err != nil {
		return "", err
	}
	resolvedPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		return "", err
	}
	return filepath.Clean(resolvedPath), nil
}

func pathWithinRoot(path, root string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

// getAllowedDirs returns the list of allowed working directories.
func (b *Bot) getAllowedDirs() []string {
	return b.config.AllowedWorkingDirs
}

// isAdmin checks if a telegram ID is an admin.
func (b *Bot) isAdmin(tid string) bool {
	for _, id := range b.config.AdminTelegramIDs {
		if strings.TrimSpace(id) == tid {
			return true
		}
	}
	return false
}

func (b *Bot) projectPathError(err error) string {
	switch {
	case errors.Is(err, errPathNotFound):
		return "Directory not found."
	case errors.Is(err, errExpectedDirectory):
		return "Path is not a directory."
	case errors.Is(err, errPathNotAllowed):
		return fmt.Sprintf("Directory not allowed. Allowed: %s", strings.Join(b.getAllowedDirs(), ", "))
	default:
		return "Failed to resolve working directory."
	}
}

func (b *Bot) filePathError(err error) string {
	switch {
	case errors.Is(err, errPathNotFound):
		return "File not found."
	case errors.Is(err, errExpectedRegularFile):
		return "Path is not a regular file."
	case errors.Is(err, errPathNotAllowed):
		return fmt.Sprintf("File not allowed. Allowed roots: %s", strings.Join(b.getAllowedDirs(), ", "))
	default:
		return "Failed to resolve file path."
	}
}

// sendFile sends a file from the server to the user.
func (b *Bot) sendFile(c tele.Context, path string) error {
	doc := &tele.Document{
		File:     tele.FromDisk(path),
		FileName: filepath.Base(path),
	}
	return c.Send(doc)
}
