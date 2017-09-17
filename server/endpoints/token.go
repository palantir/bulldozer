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

package endpoints

import (
	"context"
	"net/http"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo"
	"github.com/pkg/errors"

	"github.com/palantir/bulldozer/auth"
	gh "github.com/palantir/bulldozer/github"
	"github.com/palantir/bulldozer/log"
	"github.com/palantir/bulldozer/persist"
)

func Token(db *sqlx.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		logger := log.FromContext(c)

		token, err := auth.GithubOauthConfig.Exchange(context.TODO(), c.QueryParam("code"))
		if err != nil {
			return errors.Wrap(err, "Cannot get code from GitHub")
		}

		accessToken := token.AccessToken
		ghClient := gh.FromToken(c, accessToken)
		u, _, err := ghClient.Users.Get(ghClient.Ctx, "")
		if err != nil {
			return errors.Wrap(err, "Cannot get user from token")
		}

		user, err := persist.GetUserByID(db, u.GetID())
		if err != nil {
			dbUser := &persist.User{
				GithubID: u.GetID(),
				Name:     u.GetLogin(),
				Token:    accessToken,
			}
			if err := persist.Put(db, dbUser); err != nil {
				return errors.Wrapf(err, "Cannot add %s to the database", u.GetLogin())
			}
		} else {
			if user.Token != accessToken {
				if err := persist.UpdateUserToken(db, u.GetID(), accessToken); err != nil {
					return errors.Wrapf(err, "Cannot update token for user %s", u.GetLogin())
				}
				logger.Debugf("Updated token for user %s", u.GetLogin())
			}
		}

		p := struct {
			Result string `json:"result"`
			Token  string `json:"token"`
		}{
			Result: "ok",
			Token:  accessToken,
		}

		return c.JSON(http.StatusOK, p)
	}
}
