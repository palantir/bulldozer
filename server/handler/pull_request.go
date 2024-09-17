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

	"github.com/google/go-github/v65/github"
	"github.com/palantir/bulldozer/pull"
	"github.com/palantir/go-githubapp/githubapp"
	"github.com/pkg/errors"
)

type PullRequest struct {
	Base
}

func (h *PullRequest) Handles() []string {
	return []string{"pull_request"}
}

func (h *PullRequest) Handle(ctx context.Context, eventType, deliveryID string, payload []byte) error {
	var event github.PullRequestEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return errors.Wrap(err, "failed to parse pull request event payload")
	}

	repo := event.GetRepo()
	owner := repo.GetOwner().GetLogin()
	repoName := repo.GetName()
	number := event.GetNumber()
	installationID := githubapp.GetInstallationIDFromEvent(&event)
	ctx, logger := githubapp.PreparePRContext(ctx, installationID, repo, number)

	logger.Debug().Msgf("Received pull_request %s event", event.GetAction())

	if event.GetAction() == "closed" {
		logger.Debug().Msg("Doing nothing since pull request is closed")
		return nil
	}

	client, err := h.ClientCreator.NewInstallationClient(installationID)
	if err != nil {
		return errors.Wrap(err, "failed to instantiate github client")
	}

	pr, _, err := client.PullRequests.Get(ctx, owner, repoName, number)
	if err != nil {
		return errors.Wrapf(err, "failed to get pull request %s/%s#%d", owner, repoName, number)
	}
	pullCtx := pull.NewGithubContext(client, pr)

	config, err := h.FetchConfigForPR(ctx, client, pr)
	if err != nil {
		return err
	}

	if event.GetAction() == "labeled" || event.GetAction() == "opened" {
		if h.DisableUpdateFeature {
			logger.Debug().Msgf("Skipping updates to pull request due to server configuration override")
		} else {
			base, _ := pullCtx.Branches()
			didUpdatePR, err := h.UpdatePullRequest(logger.WithContext(ctx), pullCtx, client, config, pr, base)
			if err != nil {
				logger.Error().Err(errors.WithStack(err)).Msg("Error updating pull request")
			}

			if didUpdatePR {
				return nil
			}
		}
	}

	if err := h.ProcessPullRequest(ctx, pullCtx, client, config, pr); err != nil {
		logger.Error().Err(errors.WithStack(err)).Msg("Error processing pull request")
	}

	return nil
}

// type assertion
var _ githubapp.EventHandler = &PullRequest{}
