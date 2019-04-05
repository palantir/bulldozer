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

	"github.com/google/go-github/github"
	"github.com/palantir/go-githubapp/githubapp"
	"github.com/pkg/errors"

	"github.com/palantir/bulldozer/pull"
)

type CheckSuite struct {
	Base
}

func (h *CheckSuite) Handles() []string {
	return []string{"check_suite"}
}

func (h *CheckSuite) Handle(ctx context.Context, eventType, deliveryID string, payload []byte) error {
	var event github.CheckSuiteEvent

	if err := json.Unmarshal(payload, &event); err != nil {
		return errors.Wrap(err, "failed to parse status event payload")
	}

	repo := event.GetRepo()
	installationID := githubapp.GetInstallationIDFromEvent(&event)
	ctx, logger := githubapp.PrepareRepoContext(ctx, installationID, repo)
	logger.Debug().Msg("NEW HANDLER")
	if event.GetAction() != "completed" {
		logger.Debug().Msgf("Doing nothing since check_run action was %q instead of 'completed'", event.GetAction())
		return nil
	}
	suite := event.GetCheckSuite()

	client, err := h.ClientCreator.NewInstallationClient(installationID)
	if err != nil {
		return errors.Wrap(err, "failed to instantiate github client")
	}

	prs := suite.PullRequests

	if len(prs) == 0 {
		logger.Debug().Msg("Doing nothing since status change event affects no open pull requests")
		return nil
	}

	for _, pr := range prs {
		pullCtx := pull.NewGithubContext(client, pr)
		logger := logger.With().Int(githubapp.LogKeyPRNum, pr.GetNumber()).Logger()
		if err := h.ProcessPullRequest(logger.WithContext(ctx), pullCtx, client, pr); err != nil {
			logger.Error().Err(errors.WithStack(err)).Msg("Error processing pull request")
		}
	}

	return nil
}

// type assertion
var _ githubapp.EventHandler = &CheckSuite{}
