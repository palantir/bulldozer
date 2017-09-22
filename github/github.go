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
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/google/go-github/github"
	"github.com/labstack/echo"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v2"

	"github.com/palantir/bulldozer/log"
	"github.com/palantir/bulldozer/server/config"
	"github.com/palantir/bulldozer/utils"
)

const (
	MergeMethod  = "merge"
	SquashMethod = "squash"
	RebaseMethod = "rebase"

	PingEvent              = "ping"
	StatusEvent            = "status"
	PullRequestEvent       = "pull_request"
	PullRequestReviewEvent = "pull_request_review"
	PushEvent              = "push"

	ModeWhitelist = "whitelist"
	ModeBlacklist = "blacklist"
	ModeBody      = "pr_body"
)

var (
	AcceptedPermLevels = []string{"write", "admin"}
)

type Client struct {
	Logger *logrus.Entry
	Ctx    context.Context

	*github.Client
}

type BulldozerFile struct {
	Strategy         string `yaml:"strategy" validate:"nonzero"`
	DeleteAfterMerge bool   `yaml:"deleteAfterMerge" validate:"nonzero"`
	Mode             string `yaml:"mode" validate:"nonzero"`
}

func FromAuthHeader(c echo.Context, authHeader string) (*Client, error) {
	if authHeader == "" {
		return nil, errors.New("authorization header not present")
	}
	token := strings.Split(authHeader, " ")[1]

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{
			AccessToken: token,
		},
	)
	tc := oauth2.NewClient(context.TODO(), ts)

	client := github.NewClient(tc)

	client.BaseURL, _ = url.Parse(config.Instance.Github.APIURL)

	return &Client{log.FromContext(c), context.TODO(), client}, nil
}

func FromToken(c echo.Context, token string) *Client {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(context.TODO(), ts)

	client := github.NewClient(tc)

	client.BaseURL, _ = url.Parse(config.Instance.Github.APIURL)

	return &Client{log.FromContext(c), context.TODO(), client}
}

func (client *Client) ConfigFile(repo *github.Repository, ref string) (*BulldozerFile, error) {
	owner := repo.Owner.GetLogin()
	name := repo.GetName()

	repositoryContent, _, _, err := client.Repositories.GetContents(client.Ctx, owner, name, ".bulldozer.yml", &github.RepositoryContentGetOptions{
		Ref: ref,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "cannot get .bulldozer.yml for %s on ref %s", repo.GetFullName(), ref)
	}

	content, err := repositoryContent.GetContent()
	if err != nil {
		return nil, err
	}

	var bulldozerFile BulldozerFile
	err = yaml.Unmarshal([]byte(content), &bulldozerFile)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot unmarshal .bulldozer.yml for %s on %s", repo.GetFullName(), ref)
	}

	return &bulldozerFile, nil
}

func (client *Client) MergeMethod(branch *github.PullRequestBranch) (string, error) {
	logger := client.Logger

	owner := branch.Repo.Owner.GetLogin()
	name := branch.Repo.GetName()

	cfgFile, err := client.ConfigFile(branch.Repo, branch.GetRef())
	if err != nil {
		return "", err
	}

	repo, _, err := client.Repositories.Get(client.Ctx, owner, name)
	if err != nil {
		return "", errors.Wrapf(err, "cannot get %s", branch.Repo.GetFullName())
	}

	allowedMethods := []string{SquashMethod, MergeMethod, RebaseMethod}

	rAllowedMethods := []struct {
		name      string
		isAllowed bool
	}{
		{
			name:      SquashMethod,
			isAllowed: repo.GetAllowSquashMerge(),
		},
		{
			name:      MergeMethod,
			isAllowed: repo.GetAllowMergeCommit(),
		},
		{
			name:      RebaseMethod,
			isAllowed: repo.GetAllowRebaseMerge(),
		},
	}

	validMethod := func() bool {
		for _, method := range allowedMethods {
			if method == cfgFile.Strategy {
				return true
			}
		}

		return false
	}()

	if !validMethod {
		for _, method := range rAllowedMethods {
			if method.isAllowed {
				logger.WithFields(logrus.Fields{
					"repo":         repo.GetFullName(),
					"wantedMethod": cfgFile.Strategy,
					"chosenMethod": method.name,
				}).Debug("Wanted merge method is not allowed, fallback one was chosen")
				return method.name, nil
			}
		}
	}

	desiredIsAllowed := func() bool {
		switch cfgFile.Strategy {
		case MergeMethod:
			return repo.GetAllowMergeCommit()
		case SquashMethod:
			return repo.GetAllowSquashMerge()
		default:
			return repo.GetAllowRebaseMerge()

		}
	}()

	if desiredIsAllowed {
		return cfgFile.Strategy, nil
	}

	var m string
	for _, method := range rAllowedMethods {
		if method.isAllowed {
			m = method.name
			break
		}
	}

	return m, nil
}

func (client *Client) DeleteFlag(branch *github.PullRequestBranch) (bool, error) {
	bulldozerFile, err := client.ConfigFile(branch.Repo, branch.GetRef())
	if err != nil {
		return false, err
	}

	return bulldozerFile.DeleteAfterMerge, nil
}

func (client *Client) OperationMode(branch *github.PullRequestBranch) (string, error) {
	cfgFile, err := client.ConfigFile(branch.Repo, branch.GetRef())
	if err != nil {
		return "", err
	}

	if !utils.StringInSlice(cfgFile.Mode, []string{ModeBlacklist, ModeWhitelist, ModeBody}) {
		return "", fmt.Errorf("%s is not a valid operation mode", cfgFile.Mode)
	}

	return cfgFile.Mode, nil
}

func (client *Client) AllCommits(pr *github.PullRequest) ([]*github.RepositoryCommit, error) {
	repo := pr.Base.Repo
	owner := repo.Owner.GetLogin()
	name := repo.GetName()

	var repositoryCommits []*github.RepositoryCommit
	opts := &github.ListOptions{
		PerPage: 100,
	}

	for {
		commits, resp, err := client.PullRequests.ListCommits(client.Ctx, owner, name, pr.GetNumber(), opts)
		if err != nil {
			return nil, err
		}
		repositoryCommits = append(repositoryCommits, commits...)
		if resp.NextPage == 0 {
			break
		}

		opts.Page = resp.NextPage
	}

	return repositoryCommits, nil
}

func (client *Client) CommitMessages(pr *github.PullRequest) ([]string, error) {
	repo := pr.Base.Repo

	repositoryCommits, err := client.AllCommits(pr)
	if err != nil {
		return []string{}, errors.Wrapf(err, "cannot list commits for %s-%d", repo.GetFullName(), pr.GetNumber())
	}

	var commitMessages []string
	for _, repositoryCommit := range repositoryCommits {
		commitMessages = append(commitMessages, fmt.Sprintf("* %s", repositoryCommit.Commit.GetMessage()))
	}

	return commitMessages, nil
}

func (client *Client) Merge(pr *github.PullRequest) error {
	logger := client.Logger

	repo := pr.Base.Repo
	owner := repo.Owner.GetLogin()
	name := repo.GetName()

	commitMessage := ""
	mergeMethod, err := client.MergeMethod(pr.Base)
	if err != nil {
		return errors.Wrapf(err, "cannot get merge method for %s on ref %s", repo.GetFullName(), pr.Base.GetRef())
	}

	if mergeMethod == SquashMethod {
		messages, err := client.CommitMessages(pr)
		if err != nil {
			return err
		}
		for _, message := range messages {
			commitMessage = fmt.Sprintf("%s%s\n", commitMessage, message)
		}

		var r *regexp.Regexp
		if strings.Contains(pr.GetBody(), "==COMMIT_MSG==") {
			r = regexp.MustCompile("(?s:(==COMMIT_MSG==\r\n)(.*)(\r\n==COMMIT_MSG==))")
		} else if strings.Contains(pr.GetBody(), "==SQUASH_MSG==") {
			r = regexp.MustCompile("(?s:(==SQUASH_MSG==\r\n)(.*)(\r\n==SQUASH_MSG==))")
		}
		if r != nil {
			m := r.FindStringSubmatch(pr.GetBody())
			if len(m) == 4 {
				commitMessage = m[2]
			}
		}
	}

	delete, err := client.DeleteFlag(pr.Base)
	if err != nil {
		return errors.Wrapf(err, "cannot get delete flag for %s on ref %s", repo.GetFullName(), pr.Base.GetRef())
	}

	mergeCommit, _, err := client.PullRequests.Merge(client.Ctx, owner, name, pr.GetNumber(),
		commitMessage,
		&github.PullRequestOptions{
			MergeMethod: mergeMethod,
		})
	if err != nil {
		return errors.Wrapf(err, "merge of %s-%d has failed", repo.GetFullName(), pr.GetNumber())
	}

	logger.WithFields(logrus.Fields{
		"repo":   repo.GetFullName(),
		"pr":     pr.GetNumber(),
		"method": mergeMethod,
		"sha":    mergeCommit.GetSHA(),
	}).Info("Merged pull request")

	if !pr.Head.Repo.GetFork() {
		if delete {
			_, err = client.Git.DeleteRef(client.Ctx, owner, name, fmt.Sprintf("heads/%s", pr.Head.GetRef()))
			if err != nil {
				return errors.Wrapf(err, "cannot delete ref %s on %s", pr.Head.GetRef(), repo.GetFullName())
			}
			logger.WithFields(logrus.Fields{
				"repo": repo.GetFullName(),
				"ref":  pr.Head.GetRef(),
			}).Info("Deleted ref on repo")
		}
	} else {
		logger.WithFields(logrus.Fields{
			"repo": repo.GetFullName(),
			"pr":   pr.GetNumber(),
		}).Debug("Pull request is from a fork, not deleting if enabled")
	}

	_, _, err = client.Issues.CreateComment(client.Ctx, owner, name, pr.GetNumber(), &github.IssueComment{
		Body: github.String("Automatically merged via Bulldozer!"),
	})
	if err != nil {
		return errors.Wrapf(err, "cannot comment on %s-%d", repo.GetFullName(), pr.GetNumber())
	}

	return nil
}

func (client *Client) PullRequestForSHA(repo *github.Repository, SHA string) (*github.PullRequest, error) {
	owner := repo.Owner.GetLogin()
	name := repo.GetName()
	opt := &github.PullRequestListOptions{
		State: "open",
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}
	var pullRequest *github.PullRequest
	for {
		prs, resp, err := client.PullRequests.List(client.Ctx, owner, name, opt)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot list PRs for repository %s", repo)
		}
		for _, pr := range prs {
			if pr.Head.GetSHA() == SHA {
				pullRequest = pr
				break
			}
		}
		if resp.NextPage == 0 {
			break
		}
		opt.ListOptions.Page = resp.NextPage
	}

	if pullRequest == nil {
		return nil, nil
	}

	if pullRequest.Mergeable != nil {
		return pullRequest, nil
	}

	ticker := time.NewTicker(5 * time.Second)
	for {
		<-ticker.C

		// polling for merge status (https://developer.github.com/v3/pulls/#get-a-single-pull-request)
		p, _, err := client.PullRequests.Get(client.Ctx, owner, name, pullRequest.GetNumber())
		if err != nil {
			return nil, errors.Wrapf(err, "cannot poll for PR merge status with head %s on %s", SHA, repo.GetFullName())
		}
		if p.Mergeable != nil {
			pullRequest = p
			break
		}
	}

	return pullRequest, nil
}

func (client *Client) ReviewStatus(pr *github.PullRequest) (bool, error) {
	logger := client.Logger

	repo := pr.Base.Repo
	owner := repo.Owner.GetLogin()
	name := repo.GetName()

	reviewers, _, err := client.PullRequests.ListReviewers(client.Ctx, owner, name, pr.GetNumber(), nil)
	if err != nil {
		return false, errors.Wrapf(err, "cannot list reviewers for PR %d on %s", pr.GetNumber(), repo.GetFullName())
	}
	reviews, _, err := client.PullRequests.ListReviews(client.Ctx, owner, name, pr.GetNumber(), nil)
	if err != nil {
		return false, errors.Wrapf(err, "cannot list reviews for PR %d on %s", pr.GetNumber(), repo.GetFullName())
	}

	approval := false
	for _, review := range reviews {
		perms, _, err := client.Repositories.GetPermissionLevel(client.Ctx, owner, name, review.User.GetLogin())
		if err != nil {
			return false, errors.Wrapf(err, "cannot get permission level for %s on %s", review.User.GetLogin(), repo.GetFullName())
		}

		hasWrite := utils.StringInSlice(perms.GetPermission(), AcceptedPermLevels)
		if review.GetState() == "APPROVED" && hasWrite {
			logger.WithFields(logrus.Fields{
				"repo":     repo.GetFullName(),
				"pr":       pr.GetNumber(),
				"approver": review.User.GetLogin(),
			}).Info("Review approved")
			approval = true
			break
		}
	}

	protection, _, err := client.Repositories.GetBranchProtection(client.Ctx, owner, name, pr.Base.GetRef())
	if err != nil {
		ghErr := err.(*github.ErrorResponse)
		if ghErr.Response.StatusCode == http.StatusNotFound {
			logger.WithFields(logrus.Fields{
				"repo":   repo.GetFullName(),
				"branch": pr.Base.GetRef(),
			}).Debug("Branch does not have branch protection active")
			return true, nil
		}
	}
	reviewRequired := protection.RequiredPullRequestReviews != nil

	if len(reviewers) == 0 && len(reviews) == 0 && !reviewRequired {
		logger.WithFields(logrus.Fields{
			"repo": repo.GetFullName(),
			"pr":   pr.GetNumber(),
		}).Info("Pull request has 0 reviewers and 0 reviews, considering status true")
		return true, nil
	}

	logger.WithFields(logrus.Fields{
		"repo":       repo.GetFullName(),
		"pr":         pr.GetNumber(),
		"nReviewers": len(reviewers),
		"nReviews":   len(reviews),
	}).Info("Pull request has reviewers/reviews")
	logger.WithFields(logrus.Fields{
		"repo":           repo.GetFullName(),
		"pr":             pr.GetNumber(),
		"approvalStatus": approval,
	}).Info("Pull request has approval status")

	if reviewRequired {
		logger.WithFields(logrus.Fields{
			"repo":   repo.GetFullName(),
			"branch": pr.Base.GetRef(),
		}).Debug("Branch requires code reviews")
		return approval, nil
	} else if len(reviews) != 0 || len(reviewers) != 0 {
		return approval, nil
	}

	return true, nil
}

func (client *Client) LastReviewFromUser(pr *github.PullRequest, user *github.User) (*github.PullRequestReview, error) {
	repo := pr.Base.Repo
	owner := repo.Owner.GetLogin()
	name := repo.GetName()

	reviews, _, err := client.PullRequests.ListReviews(client.Ctx, owner, name, pr.GetNumber(), nil)
	if err != nil {
		return nil, err
	}

	for _, review := range reviews {
		if review.User.GetLogin() == user.GetLogin() {
			return review, nil
		}
	}

	return nil, nil
}

func (client *Client) HasLabels(pr *github.PullRequest, labelNames []string) (bool, error) {
	repo := pr.Base.Repo
	owner := repo.Owner.GetLogin()
	name := repo.GetName()

	if len(labelNames) == 0 {
		return false, nil
	}

	issue, _, err := client.Issues.Get(client.Ctx, owner, name, pr.GetNumber())
	if err != nil {
		return true, errors.Wrapf(err, "cannot get %s-%d", repo.GetFullName(), pr.GetNumber())
	}

	attachedLabels := []string{}
	for _, label := range issue.Labels {
		attachedLabels = append(attachedLabels, strings.ToLower(label.GetName()))
	}

	for _, label := range labelNames {
		if utils.StringInSlice(label, attachedLabels) {
			return true, nil
		}
	}

	return false, nil
}

func (client *Client) AllPullRequests(repo *github.Repository) ([]*github.PullRequest, error) {
	var pullRequests []*github.PullRequest
	owner := repo.Owner.GetLogin()
	name := repo.GetName()
	opt := &github.PullRequestListOptions{
		State: "open",
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}
	for {
		prs, resp, err := client.PullRequests.List(client.Ctx, owner, name, opt)
		if err != nil {
			return nil, err
		}
		pullRequests = append(pullRequests, prs...)
		if resp.NextPage == 0 {
			break
		}
		opt.ListOptions.Page = resp.NextPage
	}

	return pullRequests, nil
}

func (client *Client) LastStatusForContext(repo *github.Repository, SHA, context string) (string, error) {
	owner := repo.Owner.GetLogin()
	name := repo.GetName()

	statuses, _, err := client.Repositories.ListStatuses(client.Ctx, owner, name, SHA, &github.ListOptions{
		PerPage: 100,
	})
	if err != nil {
		return "", err
	}

	for _, status := range statuses {
		if context == status.GetContext() {
			return status.GetState(), nil
		}
	}

	return "", errors.New("context not found in last 100 statuses")
}

func (client *Client) ShaStatus(pr *github.PullRequest, SHA string) (bool, error) {
	logger := client.Logger

	eval := true
	repo := pr.Base.Repo
	owner := repo.Owner.GetLogin()
	name := repo.GetName()

	_, _, err := client.Repositories.GetBranchProtection(client.Ctx, owner, name, pr.Base.GetRef())
	if err != nil {
		ghErr := err.(*github.ErrorResponse)
		if ghErr.Response.StatusCode == http.StatusNotFound {
			logger.WithFields(logrus.Fields{
				"repo":   repo.GetFullName(),
				"branch": pr.Base.GetRef(),
			}).Debug("Branch protection is not set")
			return true, nil
		}

		return false, errors.Wrapf(err, "cannot get branch protection for %s/%s", repo.GetFullName(), pr.Base.GetRef())
	}

	combinedStatus, _, err := client.Repositories.GetCombinedStatus(client.Ctx, owner, name, SHA, nil)
	if err != nil {
		return false, errors.Wrapf(err, "cannot get combined status for SHA %s on %s", SHA, repo.GetFullName())
	}

	requiredStatusChecks, _, err := client.Repositories.GetRequiredStatusChecks(client.Ctx, owner, name, pr.Base.GetRef())
	if err != nil {
		return false, errors.Wrapf(err, "cannot get required status checks for %s", repo.GetFullName())
	}

	nRequiredStatuses := len(requiredStatusChecks.Contexts)
	nStatuses := len(combinedStatus.Statuses)

	logger.WithFields(logrus.Fields{
		"repo":             repo.GetFullName(),
		"pr":               pr.GetNumber(),
		"postedStatuses":   nStatuses,
		"requiredStatuses": nRequiredStatuses,
	}).Debug("Pull request overview")
	if nStatuses == 0 && nRequiredStatuses == 0 {
		return true, nil
	}

	for _, reqContext := range requiredStatusChecks.Contexts {
		lastStatus, err := client.LastStatusForContext(repo, SHA, reqContext)
		if err != nil {
			return false, errors.Wrapf(err, "cannot get last status for %s on repo %s", SHA, repo.GetFullName())
		}
		logger.WithFields(logrus.Fields{
			"repo":    repo.GetFullName(),
			"context": reqContext,
			"state":   lastStatus,
			"sha":     SHA,
		}).Debug("Context has last status on repository")
		eval = eval && (lastStatus == "success")
	}

	return eval, nil
}

func (client *Client) AllRepositories(user *github.User) ([]*github.Repository, error) {
	ownedRepos := []*github.Repository{}
	listOptions := &github.RepositoryListOptions{
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	for {
		repos, resp, err := client.Repositories.List(client.Ctx, "", listOptions)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot list repositories for user %s", user.GetLogin())
		}
		ownedRepos = append(ownedRepos, repos...)
		if resp.NextPage == 0 {
			break
		}
		listOptions.ListOptions.Page = resp.NextPage
	}

	return ownedRepos, nil
}
