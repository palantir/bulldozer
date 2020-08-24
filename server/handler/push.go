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

	"github.com/google/go-github/v32/github"
	"github.com/palantir/go-githubapp/githubapp"
	"github.com/pkg/errors"

	"github.com/palantir/bulldozer/pull"
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

	client, err := h.ClientCreator.NewInstallationClient(installationID)
	if err != nil {
		return errors.Wrap(err, "failed to instantiate github client")
	}

	prs, err := pull.ListOpenPullRequestsForRef(ctx, client, owner, repoName, baseRef)
	if err != nil {
		return errors.Wrap(err, "failed to determine open pull requests matching the push change")
	}

	logger.Debug().Msgf("received push event with base ref %s", baseRef)

	if len(prs) == 0 {
		logger.Debug().Msg("Doing nothing since push event affects no open pull requests")
		return nil
	}

	for _, pr := range prs {
		pullCtx := pull.NewGithubContext(client, pr)
		logger := logger.With().Int(githubapp.LogKeyPRNum, pr.GetNumber()).Logger()

		logger.Debug().Msgf("checking status for updated sha %s", baseRef)
		if err := h.UpdatePullRequest(logger.WithContext(ctx), pullCtx, client, pr, baseRef); err != nil {
			logger.Error().Err(errors.WithStack(err)).Msg("Error updating pull request")
		}
	}

	return nil
}

// type assertion
var _ githubapp.EventHandler = &Push{}
