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
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/go-github/github"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	gh "github.com/palantir/bulldozer/github"
	"github.com/palantir/bulldozer/log"
	"github.com/palantir/bulldozer/persist"
)

type Repository struct {
	ID          int    `json:"id"`
	Owner       string `json:"owner"`
	Name        string `json:"name"`
	IsEnabled   bool   `json:"isEnabled"`
	IsUserAdmin bool   `json:"isUserAdmin"`
	EnabledBy   string `json:"enabledBy,omitempty"`
	EnabledAt   string `json:"enabledAt,omitempty"`
}

func worker(c echo.Context, db *sqlx.DB, wg *sync.WaitGroup, repo *github.Repository, repoc chan *Repository, user *github.User, client *gh.Client) {
	logger := log.FromContext(c)
	defer wg.Done()

	var isAdmin, isEnabled bool
	var enabledBy, enabledAt string

	perm, _, err := client.Repositories.GetPermissionLevel(client.Ctx, repo.Owner.GetLogin(), repo.GetName(), user.GetLogin())

	if err != nil {
		logger.Error(errors.Wrapf(err, "cannot get permission level for %s on %s", user.GetLogin(), repo.GetFullName()))
		isAdmin = false
	} else {
		isAdmin = perm.GetPermission() == "admin"
	}

	repository, err := persist.GetRepositoryByID(db, repo.GetID())
	if err != nil {
		logger.WithFields(logrus.Fields{
			"repo": repo.GetFullName(),
		}).Error(errors.Wrap(err, "Cannot get repository from database"))
		return
	}

	if repository != nil {
		isEnabled = true
		enabledBy = repository.EnabledBy
		enabledAt = time.Unix(repository.EnabledAt, 0).Format(time.RFC3339)
	}

	repoc <- &Repository{
		ID:          repo.GetID(),
		Owner:       repo.Owner.GetLogin(),
		Name:        repo.GetName(),
		IsEnabled:   isEnabled,
		IsUserAdmin: isAdmin,
		EnabledAt:   enabledAt,
		EnabledBy:   enabledBy,
	}
}

func Repositories(db *sqlx.DB, ghAPIURL string) echo.HandlerFunc {
	return func(c echo.Context) error {
		var repositories []*Repository
		var wg sync.WaitGroup

		repoc := make(chan *Repository, 100)

		client, err := gh.FromAuthHeader(c, ghAPIURL, c.Request().Header.Get(echo.HeaderAuthorization))
		if err != nil {
			return errors.Wrap(err, "cannot create GitHub client")
		}

		user, _, err := client.Users.Get(client.Ctx, "")
		if err != nil {
			return errors.Wrap(err, "cannot get current user")
		}

		myRepositories, err := client.AllRepositories(user)
		if err != nil {
			return errors.Wrap(err, "cannot list user repositories")
		}

		wg.Add(len(myRepositories))
		for _, repo := range myRepositories {
			go worker(c, db, &wg, repo, repoc, user, client)
		}

		go func() {
			wg.Wait()
			close(repoc)
		}()

		for repo := range repoc {
			repositories = append(repositories, repo)
		}

		return c.JSON(http.StatusOK, repositories)
	}
}

func RepositoryEnable(db *sqlx.DB, ghAPIURL, webHookURL string, webHookSecret string) echo.HandlerFunc {
	return func(c echo.Context) error {
		logger := log.FromContext(c)

		client, err := gh.FromAuthHeader(c, ghAPIURL, c.Request().Header.Get(echo.HeaderAuthorization))
		if err != nil {
			return errors.Wrap(err, "cannot create GitHub client")
		}

		user, _, err := client.Users.Get(client.Ctx, "")
		if err != nil {
			return errors.Wrap(err, "cannot get current user")
		}

		owner := c.Param("owner")
		name := c.Param("name")
		repo, _, err := client.Repositories.Get(client.Ctx, owner, name)
		if err != nil {
			return errors.Wrapf(err, "cannot get %s/%s", owner, name)
		}

		perms, _, err := client.Repositories.GetPermissionLevel(client.Ctx, owner, name, user.GetLogin())
		if err != nil {
			return errors.Wrapf(err, "cannot get permission level for %s on %s", user.GetLogin(), repo.GetFullName())
		}

		if perms.GetPermission() != "admin" {
			return echo.NewHTTPError(http.StatusUnauthorized,
				fmt.Sprintf("%s does not have admin over %s", user.GetLogin(), repo.GetFullName()))
		}

		logger.WithFields(logrus.Fields{
			"repo": repo.GetFullName(),
			"user": user.GetLogin(),
		}).Debug("Creating hook on repository")

		hook, _, err := client.Repositories.CreateHook(client.Ctx, owner, name, &github.Hook{
			Name:   github.String("web"),
			URL:    github.String(webHookURL),
			Events: []string{"status", "pull_request_review", "pull_request", "push"},
			Config: map[string]interface{}{
				"name":         "bulldozer",
				"enabled_by":   user.GetLogin(),
				"enabled_at":   time.Now().UTC().Format(time.RFC3339),
				"url":          webHookURL,
				"secret":       webHookSecret,
				"content_type": "json",
			},
		})

		if err != nil {
			return errors.Wrapf(err, "cannot add hook to %s/%s via %s", owner, name, user.GetLogin())
		}

		logger.WithFields(logrus.Fields{
			"repo": repo.GetFullName(),
			"user": user.GetLogin(),
		}).Info("Created hook on repository")

		dbRepo := &persist.Repository{
			ID:        repo.GetID(),
			Name:      repo.GetFullName(),
			EnabledAt: time.Now().UTC().Unix(),
			EnabledBy: user.GetLogin(),
			HookID:    hook.GetID(),
		}

		err = persist.Put(db, dbRepo)
		if err != nil {
			_, e := client.Repositories.DeleteHook(client.Ctx, owner, name, hook.GetID())
			if e != nil {
				logger.Error(errors.Wrapf(err, "cannot delete hook on %s/%s (repo not saved to DB)", owner, name))
			}
			return errors.Wrapf(err, "cannot add %s/%s to the database", owner, name)
		}

		if err := client.CreateLabels(repo); err != nil {
			return errors.Wrapf(err, "cannot create Bulldozer issues on repo %s", repo.GetFullName())
		}

		data := struct {
			ID          int    `json:"id"`
			Owner       string `json:"owner"`
			Name        string `json:"name"`
			IsEnabled   bool   `json:"isEnabled"`
			IsUserAdmin bool   `json:"isUserAdmin"`
			EnabledBy   string `json:"enabledBy,omitempty"`
			EnabledAt   string `json:"enabledAt,omitempty"`
		}{
			ID:          repo.GetID(),
			Owner:       owner,
			Name:        name,
			IsEnabled:   true,
			IsUserAdmin: true,
			EnabledBy:   user.GetLogin(),
			EnabledAt:   time.Now().UTC().Format(time.RFC3339),
		}
		return c.JSON(http.StatusOK, data)
	}
}

func RepositoryDisable(db *sqlx.DB, ghAPIURL string) echo.HandlerFunc {
	return func(c echo.Context) error {
		logger := log.FromContext(c)

		client, err := gh.FromAuthHeader(c, ghAPIURL, c.Request().Header.Get(echo.HeaderAuthorization))
		if err != nil {
			return errors.Wrap(err, "cannot create GitHub client")
		}

		user, _, err := client.Users.Get(client.Ctx, "")
		if err != nil {
			return errors.Wrap(err, "cannot get current user from GitHub")
		}

		owner := c.Param("owner")
		name := c.Param("name")
		repo, _, err := client.Repositories.Get(client.Ctx, owner, name)
		if err != nil {
			return errors.Wrapf(err, "cannot get %s/%s from GitHub", owner, name)
		}

		perms, _, err := client.Repositories.GetPermissionLevel(client.Ctx, owner, name, user.GetLogin())
		if err != nil {
			return errors.Wrapf(err, "cannot get permission level for %s on %s", user.GetLogin(), repo.GetFullName())
		}

		if perms.GetPermission() != "admin" {
			return echo.NewHTTPError(http.StatusUnauthorized,
				fmt.Sprintf("%s does not have admin over %s", user.GetLogin(), repo.GetFullName()))
		}

		logger.WithFields(logrus.Fields{
			"repo": repo.GetFullName(),
			"user": user.GetLogin(),
		}).Debug("Deleting hook from repository")

		dbRepo, err := persist.GetRepositoryByID(db, repo.GetID())
		if err != nil {
			return errors.Wrapf(err, "cannot get repo with ID %d from database", repo.GetID())
		}
		_, err = client.Repositories.DeleteHook(client.Ctx, owner, name, dbRepo.HookID)
		if err != nil {
			return errors.Wrapf(err, "cannot delete hook %d for %s/%s via %s", owner, name, dbRepo.HookID, user.GetLogin())
		}

		logger.WithFields(logrus.Fields{
			"repo": repo.GetFullName(),
			"user": user.GetLogin(),
		}).Info("Deleted hook from repository")

		err = persist.Delete(db, dbRepo)
		if err != nil {
			return errors.Wrapf(err, "cannot remove %s/%s from database", owner, name)
		}

		return c.String(http.StatusOK, fmt.Sprintf("Disabled repository %s", repo.GetFullName()))
	}
}
