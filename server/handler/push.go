// Copyright 2018 Palantir Technologies, Inc.
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

package handler

import (
	"context"
	"encoding/json"

	"github.com/google/go-github/v67/github"
	"github.com/palantir/bulldozer/pull"
	"github.com/palantir/go-githubapp/githubapp"
	"github.com/pkg/errors"
)

type Push struct {
	Base
}

func (h *Push) Handles() []string {
	return []string{"push"}
}

func (h *Push) Handle(ctx context.Context, eventType, deliveryID string, payload []byte) error {
	var event github.PushEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return errors.Wrap(err, "failed to parse push event payload")
	}

	repo := event.GetRepo()
	owner := repo.GetOwner().GetLogin()
	repoName := repo.GetName()
	installationID := githubapp.GetInstallationIDFromEvent(&event)
	baseRef := event.GetRef()

	// todo: fixup PushEventRepository != Repository
	ghRepo := &github.Repository{
		Name: github.String(repoName),
		Owner: &github.User{
			Login: github.String(owner),
		},
	}

	ctx, logger := githubapp.PrepareRepoContext(ctx, installationID, ghRepo)
	logger.Debug().Msgf("Received push event with base ref %s", baseRef)

	// Skip any further processing of pull request updates if enabled at the server level
	if h.DisableUpdateFeature {
		logger.Debug().Msgf("Skipping updates to base ref %s due to server configuration override", baseRef)
		return nil
	}

	client, err := h.ClientCreator.NewInstallationClient(installationID)
	if err != nil {
		return errors.Wrap(err, "failed to instantiate github client")
	}

	prs, err := pull.GetAllOpenPullRequestsForRef(ctx, client.PullRequests, owner, repoName, baseRef)
	if err != nil {
		return errors.Wrap(err, "failed to determine open pull requests matching the push change")
	}
	if len(prs) == 0 {
		logger.Debug().Msgf("Doing nothing since push to %s affects no open pull requests", baseRef)
		return nil
	}

	// Fetch configuration once, since we know all PRs target the same ref
	config, err := h.FetchConfig(ctx, client, owner, repoName, baseRef)
	if err != nil {
		return err
	}

	for _, pr := range prs {
		logger := logger.With().Int(githubapp.LogKeyPRNum, pr.GetNumber()).Logger()
		logger.Debug().Msgf("Considering pull request for update")

		pullCtx := pull.NewGithubContext(client, pr)
		if _, err := h.UpdatePullRequest(logger.WithContext(ctx), pullCtx, client, config, pr, baseRef); err != nil {
			logger.Error().Err(errors.WithStack(err)).Msg("Error updating pull request")
		}
	}

	return nil
}

// type assertion
var _ githubapp.EventHandler = &Push{}
