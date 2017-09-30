package session

import (
	"github.com/antonlindstrom/pgstore"
	"github.com/gorilla/sessions"
)

type PostgresStore interface {
	Store
}

func NewPostgresStore(dbURL string, keyPairs ...[]byte) (PostgresStore, error) {
	store, err := pgstore.NewPGStore(dbURL, keyPairs...)
	if err != nil {
		return nil, err
	}
	return &postgresStore{store}, nil
}

type postgresStore struct {
	*pgstore.PGStore
}

func (p *postgresStore) Options(options Options) {
	p.PGStore.Options = &sessions.Options{
		Path:     options.Path,
		Domain:   options.Domain,
		MaxAge:   options.MaxAge,
		Secure:   options.Secure,
		HttpOnly: options.HttpOnly,
	}
}

func (p *postgresStore) MaxAge(age int) {
	p.PGStore.MaxAge(age)
}
