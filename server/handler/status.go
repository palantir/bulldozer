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

type Status struct {
	Base
}

func (h *Status) Handles() []string {
	return []string{"status"}
}

func (h *Status) Handle(ctx context.Context, eventType, deliveryID string, payload []byte) error {
	var event github.StatusEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return errors.Wrap(err, "failed to parse status event payload")
	}

	repo := event.GetRepo()
	checkName := event.GetName()
	checkState := event.GetState()

	owner := repo.GetOwner().GetLogin()
	repoName := repo.GetName()
	installationID := githubapp.GetInstallationIDFromEvent(&event)
	ctx, logger := githubapp.PrepareRepoContext(ctx, installationID, repo)

	logger.Debug().Msgf("Received status %s event from %s", checkState, checkName)

	if checkState != "success" {
		logger.Debug().Msgf("Doing nothing since context state for %q was %q", event.GetContext(), event.GetState())
		return nil
	}

	client, err := h.ClientCreator.NewInstallationClient(installationID)
	if err != nil {
		return errors.Wrap(err, "failed to instantiate github client")
	}

	prs, err := pull.ListOpenPullRequestsForSHA(ctx, client, owner, repoName, event.GetSHA())
	if err != nil {
		return errors.Wrap(err, "failed to determine open pull requests matching the status context change")
	}

	if len(prs) == 0 {
		logger.Debug().Msg("Doing nothing since status change event affects no open pull requests")
		return nil
	}

	for _, pr := range prs {
		pullCtx := pull.NewGithubContext(client, pr)
		logger := logger.With().Int(githubapp.LogKeyPRNum, pr.GetNumber()).Logger()
		config, err := h.FetchConfigForPR(ctx, client, pr)
		if err != nil {
			return err
		}

		if h.DisableUpdateFeature {
			logger.Debug().Msgf("Skipping updates to pull request due to server configuration override")
		} else {
			base, _ := pullCtx.Branches()
			didUpdatePR, err := h.UpdatePullRequest(logger.WithContext(ctx), pullCtx, client, config, pr, base)
			if err != nil {
				logger.Error().Err(errors.WithStack(err)).Msg("Error updating pull request")
			}
			if didUpdatePR {
				continue
			}
		}
		if err := h.ProcessPullRequest(logger.WithContext(ctx), pullCtx, client, config, pr); err != nil {
			logger.Error().Err(errors.WithStack(err)).Msg("Error processing pull request")
		}
	}

	return nil
}

// type assertion
var _ githubapp.EventHandler = &Status{}
