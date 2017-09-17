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

package github

import (
	"github.com/google/go-github/github"
	"github.com/labstack/echo"
	"github.com/pkg/errors"

	"github.com/palantir/bulldozer/log"
)

type ProcessResult struct {
	Merge      bool
	SHA        string
	RepoID     int
	UpdatedRef string
	Update     bool
}

func ProcessHook(c echo.Context, hookSecret string) (*ProcessResult, error) {
	logger := log.FromContext(c)

	payload, err := github.ValidatePayload(c.Request(), []byte(hookSecret))
	if err != nil {
		return nil, errors.Wrap(err, "cannot validate payload")
	}

	webHookType := github.WebHookType(c.Request())
	webHook, err := github.ParseWebHook(webHookType, payload)
	if err != nil {
		return nil, errors.Wrap(err, "cannot parse webhook")
	}

	logger.Debugf("Got a %s event", webHookType)
	switch webHookType {
	case StatusEvent:
		event := webHook.(github.StatusEvent)
		return &ProcessResult{RepoID: event.Repo.GetID(), SHA: event.GetSHA(), Merge: true}, nil
	case PullRequestEvent:
		event := webHook.(github.PullRequestEvent)
		if event.GetAction() == "closed" {
			return &ProcessResult{RepoID: event.Repo.GetID(), SHA: event.PullRequest.Head.GetSHA()}, nil
		}
		return &ProcessResult{RepoID: event.Repo.GetID(), SHA: event.PullRequest.Head.GetSHA(), Merge: true}, nil
	case PullRequestReviewEvent:
		event := webHook.(github.PullRequestReviewEvent)
		return &ProcessResult{RepoID: event.Repo.GetID(), SHA: event.PullRequest.Head.GetSHA(), Merge: true}, nil
	case PushEvent:
		event := webHook.(github.PushEvent)
		return &ProcessResult{RepoID: event.Repo.GetID(), Update: true, UpdatedRef: event.GetRef()}, nil
	case PingEvent:
		return &ProcessResult{}, nil
	}

	return nil, errors.New("unknown event, ignoring")
}
