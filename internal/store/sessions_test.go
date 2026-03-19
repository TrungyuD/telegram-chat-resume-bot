package store

import (
	"errors"
	"testing"

	"github.com/TrungyuD/telegram-chat-resume-bot/internal/platform/storage"
)

func TestSwitchSessionKeepsActiveSessionWhenTargetMissing(t *testing.T) {
	oldDataDir := storage.DataDir
	SetDataDir(t.TempDir())
	defer SetDataDir(oldDataDir)

	if err := InitDataDirs(); err != nil {
		t.Fatalf("InitDataDirs: %v", err)
	}

	tid := "user-1"
	first := &SessionMeta{
		SessionID:  "session-a",
		TelegramID: tid,
		Title:      "first",
		WorkingDir: "/tmp/a",
		IsActive:   true,
		CreatedAt:  NowUTC(),
		LastUsed:   NowUTC(),
	}
	second := &SessionMeta{
		SessionID:  "session-b",
		TelegramID: tid,
		Title:      "second",
		WorkingDir: "/tmp/b",
		IsActive:   false,
		CreatedAt:  NowUTC(),
		LastUsed:   NowUTC(),
	}
	if err := SaveSession(first); err != nil {
		t.Fatalf("SaveSession first: %v", err)
	}
	if err := SaveSession(second); err != nil {
		t.Fatalf("SaveSession second: %v", err)
	}

	err := SwitchSession(tid, "missing")
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}

	active, err := GetActiveSession(tid)
	if err != nil {
		t.Fatalf("GetActiveSession: %v", err)
	}
	if active == nil || active.SessionID != first.SessionID {
		t.Fatalf("expected active session %q, got %#v", first.SessionID, active)
	}
}

func TestSwitchSessionActivatesRequestedSession(t *testing.T) {
	oldDataDir := storage.DataDir
	SetDataDir(t.TempDir())
	defer SetDataDir(oldDataDir)

	if err := InitDataDirs(); err != nil {
		t.Fatalf("InitDataDirs: %v", err)
	}

	tid := "user-2"
	first := &SessionMeta{SessionID: "session-a", TelegramID: tid, WorkingDir: "/tmp/a", IsActive: true, CreatedAt: NowUTC(), LastUsed: NowUTC()}
	second := &SessionMeta{SessionID: "session-b", TelegramID: tid, WorkingDir: "/tmp/b", IsActive: false, CreatedAt: NowUTC(), LastUsed: NowUTC()}
	if err := SaveSession(first); err != nil {
		t.Fatalf("SaveSession first: %v", err)
	}
	if err := SaveSession(second); err != nil {
		t.Fatalf("SaveSession second: %v", err)
	}

	if err := SwitchSession(tid, second.SessionID); err != nil {
		t.Fatalf("SwitchSession: %v", err)
	}

	active, err := GetActiveSession(tid)
	if err != nil {
		t.Fatalf("GetActiveSession: %v", err)
	}
	if active == nil || active.SessionID != second.SessionID {
		t.Fatalf("expected active session %q, got %#v", second.SessionID, active)
	}
}
