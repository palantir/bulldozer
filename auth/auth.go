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

package auth

import (
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/ipfans/echo-session"
	"github.com/labstack/echo"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"

	"github.com/palantir/bulldozer/server/config"
)

func GitHubOAuth(cfg *config.GithubConfig) *oauth2.Config {
	return &oauth2.Config{
		RedirectURL:  cfg.CallbackURL,
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  cfg.Address + "/login/oauth/authorize",
			TokenURL: cfg.Address + "/login/oauth/access_token",
		},
		Scopes: []string{"repo", "read:org", "admin:repo_hook"},
	}
}

func BeginAuth(cfg *config.GithubConfig) echo.HandlerFunc {
	return func(c echo.Context) error {
		state := GenerateState()
		redirectURL := GitHubOAuth(cfg).AuthCodeURL(state)
		sess := session.Default(c)
		sess.Set("state", state)
		if err := sess.Save(); err != nil {
			return errors.Wrap(err, "cannot save session")
		}

		return c.Redirect(http.StatusTemporaryRedirect, redirectURL)
	}
}

func CompleteAuth(assetsDir string) echo.HandlerFunc {
	return func(c echo.Context) error {
		sess := session.Default(c)
		state := sess.Get("state")
		flowState := c.QueryParam("state")

		if flowState == "" {
			return c.Redirect(http.StatusMovedPermanently, "/")
		}
		if state != flowState {
			return errors.New("state does not match, possible MITM attempt")
		}

		return c.File(fmt.Sprintf("%s/index.html", assetsDir))
	}
}

func stringWithCharset(length int, charset string) string {
	seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, length)

	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}

	return string(b)
}

func GenerateState() string {
	charset := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	return stringWithCharset(10, charset)
}
