// Copyright 2017 Palantir Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"fmt"
	"math/rand"
	"net/http"
	"strings"

	"github.com/ipfans/echo-session"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/pkg/errors"

	"github.com/palantir/bulldozer/auth"
	bm "github.com/palantir/bulldozer/middleware"
	"github.com/palantir/bulldozer/server/config"
	"github.com/palantir/bulldozer/server/endpoints"
	"github.com/palantir/bulldozer/utils"
)

type Server struct {
	rest config.Rest
	e    *echo.Echo
}

func New(db *sqlx.DB, startup *config.Startup) *Server {
	e := echo.New()

	e.Use(bm.ContextMiddleware)
	e.Use(middleware.BodyLimit("6M"))
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: config.EchoLoggingFormat,
		Skipper: func(c echo.Context) bool {
			return strings.Contains(c.Request().URL.String(), "callback?code=")
		},
	}))
	e.Use(middleware.Recover())

	e.HTTPErrorHandler = utils.CustomHTTPErrorHandler

	registerEndpoints(startup, e, db)

	return &Server{startup.Server, e}
}

func registerEndpoints(startup *config.Startup, e *echo.Echo, db *sqlx.DB) {
	e.Static("/", startup.AssetDir)

	e.GET("/repositories", func(c echo.Context) error {
		return c.Redirect(http.StatusMovedPermanently, "/")
	})

	e.GET("/health", endpoints.Health())
	e.GET("/api/user/repos", endpoints.Repositories(db))

	e.GET("/api/auth/github", auth.BeginAuthHandler)
	e.GET("/login", auth.CompleteAuth(startup.AssetDir))

	e.POST("/api/repo/:owner/:name", endpoints.RepositoryEnable(db, startup.Github.WebHookURL, startup.Github.WebhookSecret))
	e.DELETE("/api/repo/:owner/:name", endpoints.RepositoryDisable(db))

	e.POST("/api/github/hook", endpoints.Hook(db, startup.Github.WebhookSecret, startup.ConfigPaths))
	e.GET("/api/auth/github/token", endpoints.Token(db))
}

func (s *Server) SetupSessionStore() error {
	var cookieSecretAuth = make([]byte, 32)
	var cookieSecretEnc = make([]byte, 32)

	if _, err := rand.Read(cookieSecretAuth); err != nil {
		return errors.Wrap(err, "cannot read rand cookie auth")
	}
	if _, err := rand.Read(cookieSecretEnc); err != nil {
		return errors.Wrap(err, "cannot read rand cookie secret")
	}

	cookieStore := session.NewCookieStore(cookieSecretAuth, cookieSecretEnc)
	s.e.Use(session.Sessions("bulldozer", cookieStore))

	return nil
}

func (s *Server) Start() error {
	return s.e.Start(fmt.Sprintf("%s:%d", s.rest.Address, s.rest.Port))
}
