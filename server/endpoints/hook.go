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

package endpoints

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/go-github/github"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	gh "github.com/palantir/bulldozer/github"
	"github.com/palantir/bulldozer/log"
	"github.com/palantir/bulldozer/persist"
	"github.com/palantir/bulldozer/server/config"
)

func Hook(db *sqlx.DB, secret string, configPaths []string) echo.HandlerFunc {
	return func(c echo.Context) error {
		logger := log.FromContext(c)

		result, err := gh.ProcessHook(c, secret)
		if err != nil {
			return errors.Wrap(err, "cannot process hook")
		}

		logger.Debugf("ProcessHook returned %+v", result)

		dbRepo, err := persist.GetRepositoryByID(db, result.RepoID)
		if err != nil {
			return errors.Wrapf(err, "cannot get repo with id %d from database", result.RepoID)
		}

		if dbRepo == nil {
			return errors.Wrapf(err, "repository with ID not enabled", result.RepoID)
		}

		user, err := persist.GetUserByName(db, dbRepo.EnabledBy)
		if err != nil {
			return errors.Wrapf(err, "cannot get user %s from database", dbRepo.EnabledBy)
		}

		ghClient := gh.FromToken(c, user.Token, gh.WithConfigPaths(configPaths))

		if !(result.Update || result.Merge) {
			return c.String(http.StatusOK, "Not taking action")
		}

		repo, _, err := ghClient.Repositories.GetByID(ghClient.Ctx, result.RepoID)
		if err != nil {
			return errors.Wrapf(err, "cannot get repo with id %d from GitHub", result.RepoID)
		}

		owner := repo.Owner.GetLogin()
		name := repo.GetName()

		if result.Update {
			updatedRef := result.UpdatedRef
			pullRequests, err := ghClient.AllPullRequests(repo)
			if err != nil {
				return errors.Wrapf(err, "cannot get all open PRs for %s", repo.GetFullName())
			}
			var updateTargets []*github.PullRequest

			for _, pr := range pullRequests {
				if fmt.Sprintf("refs/heads/%s", pr.Base.GetRef()) == updatedRef {
					updateTargets = append(updateTargets, pr)
				}
			}

			for _, pr := range updateTargets {
				repoConfig, err := ghClient.ConfigFile(repo, *pr.Base.Ref)
				if err != nil {
					return err
				}

				// Validate that we should update
				switch repoConfig.UpdateStrategy {
				case gh.UpdateStrategyLabel:
					if shouldUpdate, err := shouldUpdatePR(logger, ghClient, repo, pr); err != nil {
						return err
					} else if !shouldUpdate {
						continue
					}
				case gh.UpdateStrategyAlways:
				}

				if pr.Head.Repo.GetFork() {
					logger.WithFields(logrus.Fields{
						"repo": repo.GetFullName(),
						"pr":   pr.GetNumber(),
					}).Debug("Pull request is from a fork, cannot keep it up to date with base ref")
					continue
				}

				comparison, _, err := ghClient.Repositories.CompareCommits(ghClient.Ctx, owner, name, updatedRef, pr.Head.GetRef())
				if err != nil {
					return errors.Wrapf(err, "cannot compare %s vs %s for %s", updatedRef, pr.Head.GetRef(), repo.GetFullName())
				}
				if comparison.GetBehindBy() > 0 {
					logger.WithFields(logrus.Fields{
						"repo":     repo.GetFullName(),
						"pr":       pr.GetNumber(),
						"aheadBy":  comparison.GetAheadBy(),
						"behindBy": comparison.GetBehindBy(),
					}).Debug("Pull request is not up to date with base ref")
					mergeRequest := &github.RepositoryMergeRequest{
						Base: github.String(pr.Head.GetRef()),
						Head: github.String(updatedRef),
					}
					mergeCommit, _, err := ghClient.Repositories.Merge(ghClient.Ctx, owner, name, mergeRequest)
					if err != nil {
						return errors.Wrapf(err, "cannot merge %s into %s for %s", updatedRef, pr.Head.GetRef(), repo.GetFullName())
					}

					logger.WithFields(logrus.Fields{
						"repo":       repo.GetFullName(),
						"pr":         pr.GetNumber(),
						"updatedRef": updatedRef,
						"mergeSHA":   mergeCommit.GetSHA(),
					}).Info("Base ref has been merged into head ref of pull request")
				}
			}

			return c.String(http.StatusOK, "Updated pull requests")
		}

		pr, err := ghClient.PullRequestForSHA(repo, result.SHA)
		if err != nil {
			return errors.Wrapf(err, "cannot get PR for %s/%s", repo.GetFullName(), result.SHA)
		}
		if pr == nil {
			logger.WithFields(logrus.Fields{
				"repo": repo.GetFullName(),
				"sha":  result.SHA,
			}).Debug("SHA is not head of an open pull request")
			return c.NoContent(http.StatusOK)
		}

		mode, err := ghClient.OperationMode(pr.Base)
		if err != nil {
			return errors.Wrapf(err, "cannot get operation mode for %s/%s", repo.GetFullName(), pr.Base.GetRef())
		}

		switch mode {
		case gh.ModeBlacklist:
			hasDoNotMerge, err := ghClient.HasLabels(pr, config.DoNotMerge)
			if err != nil {
				return errors.Wrapf(err, "cannot get labels for %s-%d", repo.GetFullName(), pr.GetNumber())
			}
			if hasDoNotMerge {
				logger.WithFields(logrus.Fields{
					"repo": repo.GetFullName(),
					"pr":   pr.GetNumber(),
				}).Info("Pull request has do not merge label")
				return c.String(http.StatusOK, fmt.Sprintf("Skipping PR %d", pr.GetNumber()))
			}
		case gh.ModeWhitelist:
			hasMergeWhenReady, err := ghClient.HasLabels(pr, config.MergeWhenReady)
			if err != nil {
				return errors.Wrapf(err, "cannot get labels for %s-%d", repo.GetFullName(), pr.GetNumber())
			}
			if !hasMergeWhenReady {
				logger.Infof("%s-%d does not have merge when ready label, skipping", repo.GetFullName(), pr.GetNumber())
				logger.WithFields(logrus.Fields{
					"repo": repo.GetFullName(),
					"pr":   pr.GetNumber(),
				}).Info("Pull request doest not have merge when ready label")
				return c.String(http.StatusOK, fmt.Sprintf("Skipping PR %d", pr.GetNumber()))
			}
		case gh.ModeBody:
			if !strings.Contains(pr.GetBody(), "==MERGE_WHEN_READY==") {
				logger.WithFields(logrus.Fields{
					"repo": repo.GetFullName(),
					"pr":   pr.GetNumber(),
				}).Info("Pull request does not have ==MERGE_WHEN_READY== body content")
				return c.String(http.StatusOK, fmt.Sprintf("Skipping PR %d", pr.GetNumber()))
			}
		}

		reviewStatus, err := ghClient.ReviewStatus(pr)
		if err != nil {
			return errors.Wrapf(err, "cannot get review state for %s-%d", repo.GetFullName(), pr.GetNumber())
		}
		shaStatus, err := ghClient.ShaStatus(pr, result.SHA)
		if err != nil {
			return errors.Wrapf(err, "cannot evaluate status for %s-%s", repo.GetFullName(), result.SHA)
		}

		overallStatus := shaStatus && reviewStatus && pr.GetMergeable()

		logger.Infof("%s-%d has sha status %v, review status %v and mergeable %v", repo.GetFullName(), pr.GetNumber(), shaStatus, reviewStatus, pr.GetMergeable())

		switch overallStatus {
		case true:
			logger.Infof("Merging %s-%d", repo.GetFullName(), pr.GetNumber())
			err := ghClient.Merge(pr)
			if err != nil {
				return errors.Wrapf(err, "cannot merge %s-%d into target branch %s", repo.GetFullName(), pr.GetNumber(), pr.Base.GetRef())
			}
		default:
			msg := fmt.Sprintf("SHA %s on %s-%d has state %v, not doing anything", result.SHA, repo.GetFullName(), pr.GetNumber(), overallStatus)
			logger.Info(msg)
			return c.String(http.StatusOK, msg)
		}

		zen, _, _ := ghClient.Zen(ghClient.Ctx)
		msg := fmt.Sprintf("Action executed successfuly! Here is some zen: %s", zen)

		return c.String(http.StatusOK, msg)
	}
}

func shouldUpdatePR(logger *logrus.Entry, ghClient *gh.Client, repo *github.Repository, pr *github.PullRequest) (bool, error) {
	updateLabel, err := ghClient.HasLabels(pr, config.UpdateMe)
	if err != nil {
		return false, errors.Wrapf(err, "cannot check if %s-%d has update label", repo.GetFullName(), pr.GetNumber())
	}
	if !updateLabel {
		logger.WithFields(logrus.Fields{
			"repo": repo.GetFullName(),
			"pr":   pr.GetNumber(),
		}).Info("Pull request does not have update me label, not updating")
		return false, nil
	}
	return true, nil
}
