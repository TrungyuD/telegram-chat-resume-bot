package bot

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/TrungyuD/telegram-chat-resume-bot/internal/store"
)

func TestResolveUserPathRejectsPrefixBypass(t *testing.T) {
	rootDir := t.TempDir()
	allowedRoot := filepath.Join(rootDir, "allowed")
	prefixBypass := filepath.Join(rootDir, "allowed-evil")
	if err := os.MkdirAll(allowedRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll allowedRoot: %v", err)
	}
	if err := os.MkdirAll(prefixBypass, 0o755); err != nil {
		t.Fatalf("MkdirAll prefixBypass: %v", err)
	}

	bot := &Bot{config: &store.GlobalConfig{AllowedWorkingDirs: []string{allowedRoot}}}
	_, err := bot.resolveUserPath(prefixBypass, allowedRoot, pathTargetDirectory)
	if !errors.Is(err, errPathNotAllowed) {
		t.Fatalf("expected errPathNotAllowed, got %v", err)
	}
}

func TestResolveUserPathRejectsSymlinkEscape(t *testing.T) {
	rootDir := t.TempDir()
	allowedRoot := filepath.Join(rootDir, "allowed")
	outsideRoot := filepath.Join(rootDir, "outside")
	if err := os.MkdirAll(allowedRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll allowedRoot: %v", err)
	}
	if err := os.MkdirAll(outsideRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll outsideRoot: %v", err)
	}

	outsideFile := filepath.Join(outsideRoot, "secret.txt")
	if err := os.WriteFile(outsideFile, []byte("secret"), 0o644); err != nil {
		t.Fatalf("WriteFile outsideFile: %v", err)
	}

	linkPath := filepath.Join(allowedRoot, "escape.txt")
	if err := os.Symlink(outsideFile, linkPath); err != nil {
		t.Fatalf("Symlink: %v", err)
	}

	bot := &Bot{config: &store.GlobalConfig{AllowedWorkingDirs: []string{allowedRoot}}}
	_, err := bot.resolveUserPath(linkPath, allowedRoot, pathTargetRegularFile)
	if !errors.Is(err, errPathNotAllowed) {
		t.Fatalf("expected errPathNotAllowed, got %v", err)
	}
}

func TestResolveUserPathAcceptsAllowedRegularFile(t *testing.T) {
	rootDir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatalf("EvalSymlinks: %v", err)
	}
	allowedRoot := filepath.Join(rootDir, "allowed")
	if err := os.MkdirAll(allowedRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll allowedRoot: %v", err)
	}

	filePath := filepath.Join(allowedRoot, "note.txt")
	if err := os.WriteFile(filePath, []byte("ok"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	bot := &Bot{config: &store.GlobalConfig{AllowedWorkingDirs: []string{allowedRoot}}}
	resolved, err := bot.resolveUserPath(filePath, allowedRoot, pathTargetRegularFile)
	if err != nil {
		t.Fatalf("resolveUserPath: %v", err)
	}
	if resolved != filePath {
		t.Fatalf("expected %q, got %q", filePath, resolved)
	}
}
