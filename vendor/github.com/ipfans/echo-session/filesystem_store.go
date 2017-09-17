package session

import (
	"github.com/gorilla/sessions"
)

type FileSystemStore interface {
	Store
	MaxLength(int)
}

// The path argument is the directory where sessions will be saved. If empty
// it will use os.TempDir().
//
// Keys are defined in pairs to allow key rotation, but the common case is to set a single
// authentication key and optionally an encryption key.
//
// The first key in a pair is used for authentication and the second for encryption. The
// encryption key can be set to nil or omitted in the last pair, but the authentication key
// is required in all pairs.
//
// It is recommended to use an authentication key with 32 or 64 bytes. The encryption key,
// if set, must be either 16, 24, or 32 bytes to select AES-128, AES-192, or AES-256 modes.
func NewFileSystemStoreStore(path string, keyPairs ...[]byte) FileSystemStore {
	return &filesystemStore{sessions.NewFilesystemStore(path, keyPairs...)}
}

type filesystemStore struct {
	*sessions.FilesystemStore
}

func (s *filesystemStore) Options(options Options) {
	s.FilesystemStore.Options = &sessions.Options{
		Path:     options.Path,
		Domain:   options.Domain,
		MaxAge:   options.MaxAge,
		Secure:   options.Secure,
		HttpOnly: options.HttpOnly,
	}
}

// MaxAge sets the maximum age for the store and the underlying cookie
// implementation. Individual sessions can be deleted by setting Options.MaxAge
// = -1 for that session.
func (s *filesystemStore) MaxAge(age int) {
	s.FilesystemStore.MaxAge(age)
}

// MaxLength restricts the maximum length of new sessions to l.
// If l is 0 there is no limit to the size of a session, use with caution.
// The default for a new FilesystemStore is 4096.
func (s *filesystemStore) MaxLength(l int) {
	s.FilesystemStore.MaxLength(l)
}
