# pgstore

A session store backend for [gorilla/sessions](http://www.gorillatoolkit.org/pkg/sessions) - [src](https://github.com/gorilla/sessions).

## Installation

    make get-deps

## Documentation

Available on [godoc.org](http://www.godoc.org/github.com/antonlindstrom/pgstore).

See http://www.gorillatoolkit.org/pkg/sessions for full documentation on underlying interface.

### Example

[embedmd]:# (examples/sessions.go)
```go
package examples

import (
	"log"
	"net/http"
	"time"

	"github.com/antonlindstrom/pgstore"
)

// ExampleHandler is an example that displays the usage of PGStore.
func ExampleHandler(w http.ResponseWriter, r *http.Request) {
	// Fetch new store.
	store, err := pgstore.NewPGStore("postgres://user:password@127.0.0.1:5432/database?sslmode=verify-full", []byte("secret-key"))
	if err != nil {
		log.Fatalf(err.Error())
	}
	defer store.Close()

	// Run a background goroutine to clean up expired sessions from the database.
	defer store.StopCleanup(store.Cleanup(time.Minute * 5))

	// Get a session.
	session, err := store.Get(r, "session-key")
	if err != nil {
		log.Fatalf(err.Error())
	}

	// Add a value.
	session.Values["foo"] = "bar"

	// Save.
	if err = session.Save(r, w); err != nil {
		log.Fatalf("Error saving session: %v", err)
	}

	// Delete session.
	session.Options.MaxAge = -1
	if err = session.Save(r, w); err != nil {
		log.Fatalf("Error saving session: %v", err)
	}
}
```

## Breaking changes

* 2016-07-19 - `NewPGStore` and `NewPGStoreFromPool` now returns `(*PGStore, error)`

## Thanks

I've stolen, borrowed and gotten inspiration from the other backends available:

* [redistore](https://github.com/boj/redistore)
* [mysqlstore](https://github.com/srinathgs/mysqlstore)
* [babou dbstore](https://github.com/drbawb/babou/blob/master/lib/session/dbstore.go)

Thank you all for sharing your code!

What makes this backend different is that it's for PostgreSQL.

We've recently refactored this backend to use the standard database/sql driver instead of Gorp. This removes a dependency and makes this package very lightweight and makes database interactions very transparent. Lastly, from the standpoint of unit testing where you want to mock the database layer instead of requiring a real database, you can now easily use a package like [go-SQLMock](https://github.com/DATA-DOG/go-sqlmock) to do just that.
