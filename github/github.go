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
	"github.com/palantir/bulldozer/version"
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

	MaxPullRequestPollCount = 5
)

var (
	AcceptedPermLevels = []string{"write", "admin"}
	DefaultConfigPaths = []string{".bulldozer.yml"}
)

type UpdateStrategy string

const (
	// UpdateStrategyLabel the default value for UpdateStrategy
	UpdateStrategyLabel UpdateStrategy = "label"

	// Future feature, see https://github.com/palantir/bulldozer/issues/21
	// UpdateStrategyOnRequiredChecksPassing UpdateStrategy = "onRequiredChecksPassing"

	UpdateStrategyAlways UpdateStrategy = "always"
)

type Option func(c *Client)

func WithConfigPaths(paths []string) Option {
	return func(c *Client) {
		c.configPaths = paths
	}
}

type Client struct {
	Logger *logrus.Entry
	Ctx    context.Context

	*github.Client

	configPaths []string
}

type BulldozerFile struct {
	MergeStrategy          string         `yaml:"strategy" validate:"nonzero"`
	DeleteAfterMerge       bool           `yaml:"deleteAfterMerge" validate:"nonzero"`
	Mode                   string         `yaml:"mode" validate:"nonzero"`
	UpdateStrategy         UpdateStrategy `yaml:"updateStrategy"`
	IgnoreSquashedMessages bool           `yaml:"ignoreSquashedMessages"`
}

func FromToken(c echo.Context, token string, opts ...Option) *Client {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(context.TODO(), ts)

	gh := github.NewClient(tc)

	gh.BaseURL, _ = url.Parse(config.Instance.Github.APIURL)
	gh.UserAgent = "bulldozer/" + version.Version()

	client := &Client{
		Logger: log.FromContext(c),
		Ctx:    context.TODO(),
		Client: gh,
	}

	for _, opt := range opts {
		opt(client)
	}

	if len(client.configPaths) == 0 {
		client.configPaths = DefaultConfigPaths
	}

	return client
}

func FromAuthHeader(c echo.Context, authHeader string, opts ...Option) (*Client, error) {
	if authHeader == "" {
		return nil, errors.New("authorization header not present")
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 {
		return nil, errors.New("incorrectly formatted auth header")
	}

	token := parts[1]
	return FromToken(c, token, opts...), nil
}

func (client *Client) ConfigFile(repo *github.Repository, ref string) (*BulldozerFile, error) {
	repositoryContent, err := client.findConfigFile(repo, ref)
	if err != nil {
		return nil, err
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

	// Default update strategy
	if bulldozerFile.UpdateStrategy == "" {
		bulldozerFile.UpdateStrategy = UpdateStrategyLabel
	}

	allowedUpdateStrategies := []UpdateStrategy{UpdateStrategyAlways, UpdateStrategyLabel}
	validStrategy := func() bool {
		for _, valid := range allowedUpdateStrategies {
			if bulldozerFile.UpdateStrategy == valid {
				return true
			}
		}
		return false
	}()
	if !validStrategy {
		return nil, errors.Errorf("Invalid update strategy: %#v, valid strategies: %#v", bulldozerFile.UpdateStrategy, allowedUpdateStrategies)
	}

	return &bulldozerFile, nil
}

func (client *Client) findConfigFile(repo *github.Repository, ref string) (*github.RepositoryContent, error) {
	owner := repo.Owner.GetLogin()
	name := repo.GetName()

	opts := github.RepositoryContentGetOptions{
		Ref: ref,
	}

	for _, p := range client.configPaths {
		content, _, _, err := client.Repositories.GetContents(client.Ctx, owner, name, p, &opts)
		if err != nil {
			if rerr, ok := err.(*github.ErrorResponse); ok && rerr.Response.StatusCode == http.StatusNotFound {
				continue
			}
			return nil, errors.Wrapf(err, "cannot get %s for %s on ref %s", p, repo.GetFullName(), ref)
		}
		return content, nil
	}

	return nil, errors.Errorf("no configuration found for %s on ref %s", repo.GetFullName(), ref)
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
			if method == cfgFile.MergeStrategy {
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
					"wantedMethod": cfgFile.MergeStrategy,
					"chosenMethod": method.name,
				}).Debug("Wanted merge method is not allowed, fallback one was chosen")
				return method.name, nil
			}
		}
	}

	desiredIsAllowed := func() bool {
		switch cfgFile.MergeStrategy {
		case MergeMethod:
			return repo.GetAllowMergeCommit()
		case SquashMethod:
			return repo.GetAllowSquashMerge()
		default:
			return repo.GetAllowRebaseMerge()

		}
	}()

	if desiredIsAllowed {
		return cfgFile.MergeStrategy, nil
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

func (client *Client) IgnoreSquashedMessages(branch *github.PullRequestBranch) (bool, error) {
	bulldozerFile, err := client.ConfigFile(branch.Repo, branch.GetRef())
	if err != nil {
		return false, err
	}

	return bulldozerFile.IgnoreSquashedMessages, nil
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

func (client *Client) commitMessage(pr *github.PullRequest, mergeMethod string) (string, error) {
	commitMessage := ""
	if mergeMethod == SquashMethod {
		repo := pr.Base.Repo
		ignoreSquashedMessages, err := client.IgnoreSquashedMessages(pr.Base)
		if err != nil {
			return "", errors.Wrapf(err,
				"cannot get ignore squash messages flag for %s on ref %s",
				repo.GetFullName(),
				pr.Base.GetRef())
		}

		if !ignoreSquashedMessages {
			messages, err := client.CommitMessages(pr)
			if err != nil {
				return "", err
			}
			for _, message := range messages {
				commitMessage = fmt.Sprintf("%s%s\n", commitMessage, message)
			}
		}

		if msg, ok := extractMessageOverride(pr.GetBody()); ok {
			commitMessage = msg
		}
	}

	return commitMessage, nil
}

func extractMessageOverride(body string) (msg string, found bool) {
	var r *regexp.Regexp
	if strings.Contains(body, "==COMMIT_MSG==") {
		r = regexp.MustCompile(`(?sm:(==COMMIT_MSG==\s*)^(.*)$(\s*==COMMIT_MSG==))`)
	} else if strings.Contains(body, "==SQUASH_MSG==") {
		r = regexp.MustCompile(`(?sm:(==SQUASH_MSG==\s*)^(.*)$(\s*==SQUASH_MSG==))`)
	}

	if r != nil {
		m := r.FindStringSubmatch(body)
		if len(m) == 4 {
			msg = strings.TrimSpace(m[2])
			found = true
		}
	}
	return
}

func (client *Client) Merge(pr *github.PullRequest) error {
	logger := client.Logger

	repo := pr.Base.Repo
	owner := repo.Owner.GetLogin()
	name := repo.GetName()

	mergeMethod, err := client.MergeMethod(pr.Base)
	if err != nil {
		return errors.Wrapf(err, "cannot get merge method for %s on ref %s", repo.GetFullName(), pr.Base.GetRef())
	}
	commitMessage, err := client.commitMessage(pr, mergeMethod)
	if err != nil {
		return err
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

	if pullRequest == nil || pullRequest.Mergeable != nil {
		return pullRequest, nil
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	// polling for merge status (https://developer.github.com/v3/pulls/#get-a-single-pull-request)
	for i := 0; i < MaxPullRequestPollCount; i++ {
		<-ticker.C

		p, _, err := client.PullRequests.Get(client.Ctx, owner, name, pullRequest.GetNumber())
		if err != nil {
			return nil, errors.Wrapf(err, "cannot get details for PR %d on %s", pullRequest.GetNumber(), repo.GetFullName())
		}

		if p.GetState() != "open" || p.Mergeable != nil {
			return p, nil
		}

		pullRequest = p
	}

	client.Logger.WithFields(logrus.Fields{
		"repo": repo.GetFullName(),
		"pr":   pullRequest.GetNumber(),
	}).Warnf("Failed to get a non-nil mergeable value after %d attempts; continuing with nil", MaxPullRequestPollCount)

	return pullRequest, nil
}

func (client *Client) ReviewStatus(pr *github.PullRequest) (bool, error) {
	logger := client.Logger

	repo := pr.Base.Repo
	owner := repo.Owner.GetLogin()
	name := repo.GetName()

	reviewers, err := client.AllReviewers(pr)
	if err != nil {
		return false, err
	}

	reviews, err := client.AllReviews(pr)
	if err != nil {
		return false, err
	}

	approval := false
	disapproval := false
	for _, review := range reviews {
		perms, _, err := client.Repositories.GetPermissionLevel(client.Ctx, owner, name, review.User.GetLogin())
		if err != nil {
			return false, errors.Wrapf(err, "cannot get permission level for %s on %s", review.User.GetLogin(), repo.GetFullName())
		}

		hasWrite := utils.StringInSlice(perms.GetPermission(), AcceptedPermLevels)
		if hasWrite {
			if review.GetState() == "APPROVED" {
				logger.WithFields(logrus.Fields{
					"repo":     repo.GetFullName(),
					"pr":       pr.GetNumber(),
					"approver": review.User.GetLogin(),
				}).Info("Review approved")
				approval = true
			} else if review.GetState() == "CHANGES_REQUESTED" {
				logger.WithFields(logrus.Fields{
					"repo":     repo.GetFullName(),
					"pr":       pr.GetNumber(),
					"approver": review.User.GetLogin(),
				}).Info("Review not approved")
				disapproval = true
			}
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
	if !reviewRequired && !disapproval {
		logger.WithFields(logrus.Fields{
			"repo": repo.GetFullName(),
			"pr":   pr.GetNumber(),
		}).Info("Review not required and no one has disapproved, considering status true")
		return true, nil
	}

	logger.WithFields(logrus.Fields{
		"repo":       repo.GetFullName(),
		"pr":         pr.GetNumber(),
		"nReviewers": len(reviewers.Users),
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
	} else if len(reviews) != 0 || len(reviewers.Users) != 0 {
		return approval, nil
	}

	return true, nil
}

func (client *Client) LastReviewFromUser(pr *github.PullRequest, user *github.User) (*github.PullRequestReview, error) {
	reviews, err := client.AllReviews(pr)
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
	opts := &github.RepositoryListOptions{
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	var allRepos []*github.Repository
	for {
		repos, resp, err := client.Repositories.List(client.Ctx, "", opts)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot list repositories for user %s", user.GetLogin())
		}
		allRepos = append(allRepos, repos...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allRepos, nil
}

func (client *Client) AllReviews(pr *github.PullRequest) ([]*github.PullRequestReview, error) {
	repo := pr.Base.Repo
	owner := repo.Owner.GetLogin()
	name := repo.GetName()

	opts := &github.ListOptions{
		PerPage: 100,
	}

	var allReviews []*github.PullRequestReview
	for {
		reviews, resp, err := client.PullRequests.ListReviews(client.Ctx, owner, name, pr.GetNumber(), opts)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot list reviews for %s#%d", repo.GetFullName(), pr.GetNumber())
		}

		allReviews = append(allReviews, reviews...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allReviews, nil
}

func (client *Client) AllReviewers(pr *github.PullRequest) (*github.Reviewers, error) {
	repo := pr.Base.Repo
	owner := repo.Owner.GetLogin()
	name := repo.GetName()

	opts := &github.ListOptions{
		PerPage: 100,
	}

	var allReviewers github.Reviewers
	for {
		reviewers, resp, err := client.PullRequests.ListReviewers(client.Ctx, owner, name, pr.GetNumber(), opts)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot list reviewers for %s#%d", repo.GetFullName(), pr.GetNumber())
		}

		allReviewers.Users = append(allReviewers.Users, reviewers.Users...)
		allReviewers.Teams = append(allReviewers.Teams, reviewers.Teams...)

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return &allReviewers, nil
}
