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

package pull

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/go-github/github"
	"github.com/pkg/errors"
)

// GithubContext is a Context implementation that gets information from GitHub.
// A new instance must be created for each request.
type GithubContext struct {
	client *github.Client

	owner  string
	repo   string
	number int
	pr     *github.PullRequest

	// cached fields
	comments         []string
	requiredStatuses []string
	successStatuses  []string
}

func NewGithubContext(client *github.Client, pr *github.PullRequest, owner, repo string, number int) Context {
	return &GithubContext{
		client: client,

		pr:     pr,
		owner:  owner,
		repo:   repo,
		number: number,
	}
}

func (ghc *GithubContext) Owner() string {
	return ghc.owner
}

func (ghc *GithubContext) Repo() string {
	return ghc.repo
}

func (ghc *GithubContext) Number() int {
	return ghc.number
}

func (ghc *GithubContext) Locator() string {
	return fmt.Sprintf("%s/%s#%d", ghc.owner, ghc.repo, ghc.number)
}

func (ghc *GithubContext) Title(ctx context.Context) (string, error) {
	return ghc.pr.GetTitle(), nil
}

func (ghc *GithubContext) Body(ctx context.Context) (string, error) {
	return ghc.pr.GetBody(), nil
}

func (ghc *GithubContext) Comments(ctx context.Context) ([]string, error) {
	if ghc.comments == nil {

		prCommentOpts := &github.PullRequestListCommentsOptions{ListOptions: github.ListOptions{PerPage: 100}}
		for {
			comments, res, err := ghc.client.PullRequests.ListComments(ctx, ghc.owner, ghc.repo, ghc.number, prCommentOpts)
			if err != nil {
				return nil, errors.Wrap(err, "failed to list pull request comments")
			}

			for _, c := range comments {
				ghc.comments = append(ghc.comments, c.GetBody())
			}

			if res.NextPage == 0 {
				break
			}
			prCommentOpts.Page = res.NextPage
		}

		issueCommentOpts := &github.IssueListCommentsOptions{ListOptions: github.ListOptions{PerPage: 100}}
		for {
			comments, res, err := ghc.client.Issues.ListComments(ctx, ghc.owner, ghc.repo, ghc.number, issueCommentOpts)
			if err != nil {
				return nil, errors.Wrap(err, "failed to list issue comments")
			}

			for _, c := range comments {
				ghc.comments = append(ghc.comments, c.GetBody())
			}

			if res.NextPage == 0 {
				break
			}
			issueCommentOpts.Page = res.NextPage
		}
	}

	return ghc.comments, nil
}

func (ghc *GithubContext) Commits(ctx context.Context) ([]*Commit, error) {
	opts := &github.ListOptions{
		PerPage: 100,
	}

	var allCommits []*github.RepositoryCommit
	for {
		commits, resp, err := ghc.client.PullRequests.ListCommits(ctx, ghc.owner, ghc.repo, ghc.number, opts)
		if err != nil {
			return nil, errors.Wrap(err, "failed to list pull request commits")
		}
		allCommits = append(allCommits, commits...)
		if resp.NextPage == 0 {
			break
		}

		opts.Page = resp.NextPage
	}

	commits := make([]*Commit, len(allCommits))
	for i, c := range allCommits {
		commits[i] = &Commit{
			SHA:     c.GetCommit().GetSHA(),
			Message: c.GetCommit().GetMessage(),
		}
	}
	return commits, nil
}

func (ghc *GithubContext) RequiredStatuses(ctx context.Context) ([]string, error) {
	if ghc.requiredStatuses == nil {
		requiredStatuses, _, err := ghc.client.Repositories.GetRequiredStatusChecks(ctx, ghc.owner, ghc.repo, ghc.pr.GetBase().GetRef())
		if err != nil {
			if isNotFound(err) {
				// Github returns 404 when there are no branch protections
				return nil, nil
			}
			return ghc.requiredStatuses, errors.Wrapf(err, "cannot get required status checks for %s", ghc.Locator())
		}
		ghc.requiredStatuses = requiredStatuses.Contexts
	}

	return ghc.requiredStatuses, nil
}

func isNotFound(err error) bool {
	rerr, ok := err.(*github.ErrorResponse)
	return ok && rerr.Response.StatusCode == http.StatusNotFound
}

func (ghc *GithubContext) CurrentSuccessStatuses(ctx context.Context) ([]string, error) {
	if ghc.successStatuses == nil {
		opts := &github.ListOptions{PerPage: 100}
		var successStatuses []string

		for {
			combinedStatus, res, err := ghc.client.Repositories.GetCombinedStatus(ctx, ghc.owner, ghc.repo, ghc.pr.GetHead().GetSHA(), opts)
			if err != nil {
				return ghc.successStatuses, errors.Wrapf(err, "cannot get combined status for SHA %s on %s", ghc.pr.GetHead().GetSHA(), ghc.Locator())
			}

			for _, s := range combinedStatus.Statuses {
				if s.GetState() == "success" {
					successStatuses = append(successStatuses, s.GetContext())
				}
			}

			if res.NextPage == 0 {
				break
			}
			opts.Page = res.NextPage
		}

		ghc.successStatuses = successStatuses
	}

	return ghc.successStatuses, nil
}

func (ghc *GithubContext) Branches(ctx context.Context) (base string, head string, err error) {
	base = ghc.pr.GetBase().GetRef()

	// Check for forks
	if ghc.pr.GetHead().GetRepo().GetID() == ghc.pr.GetBase().GetRepo().GetID() {
		head = ghc.pr.GetHead().GetRef()
	} else {
		head = ghc.pr.GetHead().GetLabel()
	}

	return
}

func (ghc *GithubContext) Labels(ctx context.Context) ([]string, error) {
	var labelNames []string
	for _, label := range ghc.pr.Labels {
		labelNames = append(labelNames, label.GetName())
	}
	return labelNames, nil
}

// type assertion
var _ Context = &GithubContext{}
