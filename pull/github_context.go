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

	"github.com/google/go-github/v68/github"
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
	commits          []*Commit
	branchProtection *github.Protection
	successStatuses  []string
}

func NewGithubContext(client *github.Client, pr *github.PullRequest) Context {
	return &GithubContext{
		client: client,

		pr:     pr,
		owner:  pr.GetBase().GetRepo().GetOwner().GetLogin(),
		repo:   pr.GetBase().GetRepo().GetName(),
		number: pr.GetNumber(),
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

func (ghc *GithubContext) Title() string {
	return ghc.pr.GetTitle()
}

func (ghc *GithubContext) Body() string {
	return ghc.pr.GetBody()
}

func (ghc *GithubContext) HeadSHA() string {
	return ghc.pr.GetHead().GetSHA()
}

func (ghc *GithubContext) MergeState(ctx context.Context) (*MergeState, error) {
	pr, _, err := ghc.client.PullRequests.Get(ctx, ghc.owner, ghc.repo, ghc.number)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get pull request merge state")
	}

	return &MergeState{
		Closed:    pr.GetState() == "closed",
		Mergeable: pr.Mergeable,
	}, nil
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
	if ghc.commits == nil {
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

		ghc.commits = make([]*Commit, len(allCommits))
		for i, c := range allCommits {
			ghc.commits[i] = &Commit{
				SHA:     c.GetCommit().GetSHA(),
				Message: c.GetCommit().GetMessage(),
			}
		}
	}
	return ghc.commits, nil
}

func (ghc *GithubContext) RequiredStatuses(ctx context.Context) ([]string, error) {
	if ghc.branchProtection == nil {
		if err := ghc.loadBranchProtection(ctx); err != nil {
			return nil, err
		}
	}
	if checks := ghc.branchProtection.GetRequiredStatusChecks(); checks != nil {
		return checks.GetContexts(), nil
	}
	return nil, nil
}

func (ghc *GithubContext) PushRestrictions(ctx context.Context) (bool, error) {
	if ghc.branchProtection == nil {
		if err := ghc.loadBranchProtection(ctx); err != nil {
			return false, err
		}
	}
	if r := ghc.branchProtection.GetRestrictions(); r != nil {
		return len(r.Users) > 0 || len(r.Teams) > 0, nil
	}
	return false, nil
}

func (ghc *GithubContext) loadBranchProtection(ctx context.Context) error {
	protection, _, err := ghc.client.Repositories.GetBranchProtection(ctx, ghc.owner, ghc.repo, ghc.pr.GetBase().GetRef())
	if err != nil {
		if isNotFound(err) || err == github.ErrBranchNotProtected {
			ghc.branchProtection = &github.Protection{}
			return nil
		}
		return errors.Wrapf(err, "cannot get branch protection for %s", ghc.Locator())
	}
	ghc.branchProtection = protection
	return nil
}

func isNotFound(err error) bool {
	rerr, ok := err.(*github.ErrorResponse)
	return ok && rerr.Response.StatusCode == http.StatusNotFound
}

func (ghc *GithubContext) CurrentSuccessStatuses(ctx context.Context) ([]string, error) {
	if ghc.successStatuses == nil {
		opts := &github.ListOptions{PerPage: 100}
		var successStatuses []string
		allowedCheckConclusions := map[string]bool{
			"success": true,
			"neutral": true,
			"skipped": true,
		}

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

		checkOpts := &github.ListCheckRunsOptions{ListOptions: github.ListOptions{PerPage: 100}}
		for {
			checkRuns, res, err := ghc.client.Checks.ListCheckRunsForRef(ctx, ghc.owner, ghc.repo, ghc.pr.GetHead().GetSHA(), checkOpts)
			if err != nil {
				return ghc.successStatuses, errors.Wrapf(err, "cannot get check runs for SHA %s on %s", ghc.pr.GetHead().GetSHA(), ghc.Locator())
			}

			for _, s := range checkRuns.CheckRuns {
				if allowedCheckConclusions[s.GetConclusion()] {
					successStatuses = append(successStatuses, s.GetName())
				}
			}

			if res.NextPage == 0 {
				break
			}
			checkOpts.Page = res.NextPage
		}

		ghc.successStatuses = successStatuses
	}

	return ghc.successStatuses, nil
}

func (ghc *GithubContext) Branches() (base string, head string) {
	base = ghc.pr.GetBase().GetRef()

	// if the repository is a fork, use label to include the owner prefix
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

func (ghc *GithubContext) IsTargeted(ctx context.Context) (bool, error) {
	ref := fmt.Sprintf("refs/heads/%s", ghc.pr.GetHead().GetRef())

	prs, err := GetAllOpenPullRequestsForRef(ctx, ghc.client.PullRequests, ghc.owner, ghc.repo, ref)
	if err != nil {
		return false, errors.Wrap(err, "failed to determine targeted status")
	}
	return len(prs) > 0, nil
}

func (ghc *GithubContext) IsDraft(ctx context.Context) bool {
	return ghc.pr.GetDraft()
}

func (ghc *GithubContext) AutoMerge(ctx context.Context) bool {
	autoMerge := ghc.pr.GetAutoMerge()

	return autoMerge != nil
}

// type assertion
var _ Context = &GithubContext{}
