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

type CheckRun struct {
	Base
}

func (h *CheckRun) Handles() []string {
	return []string{"check_run"}
}

func (h *CheckRun) Handle(ctx context.Context, eventType, deliveryID string, payload []byte) error {
	var event github.CheckRunEvent

	if err := json.Unmarshal(payload, &event); err != nil {
		return errors.Wrap(err, "failed to parse check_run event payload")
	}

	repo := event.GetRepo()
	owner := repo.GetOwner().GetLogin()
	repoName := repo.GetName()

	installationID := githubapp.GetInstallationIDFromEvent(&event)

	ctx, logger := githubapp.PrepareRepoContext(ctx, installationID, repo)

	if event.GetAction() != "completed" {
		logger.Debug().Msgf("Doing nothing since check_run action was %q instead of 'completed'", event.GetAction())
		return nil
	}

	client, err := h.ClientCreator.NewInstallationClient(installationID)
	if err != nil {
		return errors.Wrap(err, "failed to instantiate github client")
	}

	prs := event.GetCheckRun().PullRequests
	if len(prs) == 0 {
		logger.Debug().Msg("No pull requests associated with the check run, searching by SHA")
		// check runs on fork PRs do not have the PRs attached to the event so we need to filter all PRs by SHA
		prs, err = pull.ListAllOpenPullRequestsFilteredBySHA(ctx, client.PullRequests, owner, repoName, event.GetCheckRun().GetHeadSHA())
		if err != nil {
			return errors.Wrap(err, "failed to determine open pull requests matching the status context change")
		}
		if len(prs) == 0 {
			logger.Debug().Msg("No open pull requests found for the given SHA")
			return nil
		}
	}

	for _, pr := range prs {
		logger := logger.With().Int(githubapp.LogKeyPRNum, pr.GetNumber()).Logger()
		ctx := logger.WithContext(ctx)

		// The PR included in the CheckRun response is very slim on information.
		// It does not contain the owner information or label information we
		// need to process the pull request.
		fullPR, _, err := client.PullRequests.Get(ctx, repo.GetOwner().GetLogin(), repo.GetName(), pr.GetNumber())
		if err != nil {
			return errors.Wrapf(err, "failed to fetch PR number %q for CheckRun", pr.GetNumber())
		}
		pullCtx := pull.NewGithubContext(client, fullPR)

		config, err := h.FetchConfigForPR(ctx, client, fullPR)
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
		if err := h.ProcessPullRequest(ctx, pullCtx, client, config, fullPR); err != nil {
			logger.Error().Err(errors.WithStack(err)).Msg("Error processing pull request")
		}
	}

	return nil
}

// type assertion
var _ githubapp.EventHandler = &CheckRun{}
