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
	"time"

	"github.com/google/go-github/v41/github"
	"github.com/palantir/bulldozer/bulldozer"
	"github.com/palantir/bulldozer/pull"
	"github.com/palantir/go-githubapp/githubapp"
	"github.com/pkg/errors"
)

const (
	PullQueryDelay = 250 * time.Millisecond

	PullUpdateBaseDelay = 1 * time.Second
	PullUpdateMaxDelay  = 60 * time.Second
	PullUpdateDelayMult = 1.5
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

	client, err := h.ClientCreator.NewInstallationClient(installationID)
	if err != nil {
		return errors.Wrap(err, "failed to instantiate github client")
	}

	prs, err := pull.ListOpenPullRequestsForRef(ctx, client, owner, repoName, baseRef)
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

	if config == nil {
		logger.Debug().Msg("Skipping pull request updates to missing configuration")
		return nil
	}

	var toUpdate []updateCtx
	for i, pr := range prs {
		logger := logger.With().Int(githubapp.LogKeyPRNum, pr.GetNumber()).Logger()
		logger.Debug().Msg("Checking if pull request should update")

		ctx := logger.WithContext(ctx)
		pullCtx := pull.NewGithubContext(client, pr)

		shouldUpdate, err := bulldozer.ShouldUpdatePR(ctx, pullCtx, config.Update)
		if err != nil {
			logger.Error().Err(err).Msg("Error determining if pull request should update, skipping")
			continue
		}
		if shouldUpdate {
			toUpdate = append(toUpdate, updateCtx{
				ctx:     ctx,
				pullCtx: pullCtx,
			})
		}

		if i < len(prs)-1 {
			time.Sleep(delay(i, PullQueryDelay, 1, PullQueryDelay))
		}
	}

	logger.Info().Msgf("Found %d pull requests that need updates", len(toUpdate))
	for i, pr := range toUpdate {
		bulldozer.UpdatePR(pr.ctx, pr.pullCtx, client, config.Update, baseRef)
		if i < len(toUpdate)-1 {
			d := delay(i, PullUpdateBaseDelay, PullUpdateDelayMult, PullUpdateMaxDelay)
			logger.Debug().Msgf("Waiting %v until next update to avoid GitHub rate limits", d)
			time.Sleep(d)
		}
	}

	return nil
}

type updateCtx struct {
	ctx     context.Context
	pullCtx pull.Context
}

func delay(iter int, base time.Duration, mult float64, max time.Duration) time.Duration {
	t := base
	for ; iter > 0; iter-- {
		t = time.Duration(mult * float64(t))
		if t > max {
			return max
		}
	}
	return t
}

// type assertion
var _ githubapp.EventHandler = &Push{}
