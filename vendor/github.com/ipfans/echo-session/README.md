echo-session
======

Middleware echo-session is a session support for [echo](https://github.com/labstack/echo/).

### Installation

	go get github.com/ipfans/echo-session

## Example

```go
package main

import (
	"github.com/ipfans/echo-session"
	"github.com/labstack/echo"
	"github.com/labstack/echo/engine/standard"
	"github.com/labstack/echo/middleware"
)

func main() {
	serv := echo.New()
	serv.Use(middleware.Logger())
	serv.Use(middleware.Recover())
	// store := session.NewCookieStore([]byte("secret"))
	store, err := session.NewRedisStore(32, "tcp", "localhost:6379", "", []byte("secret"))
	if err != nil {
		panic(err)
	}
	serv.Use(session.Sessions("GSESSION", store))
	serv.Get("/", func(ctx echo.Context) error {
		session := session.Default(ctx)
		var count int
		v := session.Get("count")
		if v == nil {
			count = 0
		} else {
			count = v.(int)
			count += 1
		}
		session.Set("count", count)
		session.Save()
		ctx.JSON(200, map[string]interface{}{
			"visit": count,
		})
		return nil
	})
	serv.Run(standard.New(":8080"))
}

```

## License

This project is under Apache v2 License. See the [LICENSE](LICENSE) file for the full license text.
