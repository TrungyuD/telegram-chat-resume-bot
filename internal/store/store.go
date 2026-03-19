// Package store provides backward-compatible access to storage primitives
// and domain repositories. New code should import domain packages directly.
package store

import (
	"github.com/TrungyuD/telegram-chat-resume-bot/internal/platform/storage"
)

// GetDataDir returns the current data directory from the storage package.
func GetDataDir() string {
	return storage.DataDir
}

// SetDataDir updates the data directory in the storage package.
// This is primarily for test isolation. Setting store.DataDir directly
// has no effect; use this function instead.
func SetDataDir(dir string) {
	storage.SetDataDir(dir)
}

// Re-export types from storage for backward compatibility.
type SessionMeta = storage.SessionMeta
type SessionMessage = storage.SessionMessage

// Re-export functions from storage for backward compatibility.
var (
	LockFile                 = storage.LockFile
	EnsureDir                = storage.EnsureDir
	WriteJSON                = storage.WriteJSON
	DeleteFile               = storage.DeleteFile
	ListJSONFiles            = storage.ListJSONFiles
	ListMDFiles              = storage.ListMDFiles
	ListSubDirs              = storage.ListSubDirs
	SafeFilename             = storage.SafeFilename
	FileExists               = storage.FileExists
	NowUTC                   = storage.NowUTC
	ParseSessionMD           = storage.ParseSessionMD
	WriteSessionMD           = storage.WriteSessionMD
	AppendSessionMessage     = storage.AppendSessionMessage
	UpdateSessionFrontmatter = storage.UpdateSessionFrontmatter
	InitDataDirs             = storage.InitDataDirs
)

// ReadJSON re-exported as a wrapper because Go cannot alias generic functions via var.
func ReadJSON[T any](path string) (T, error) {
	return storage.ReadJSON[T](path)
}
