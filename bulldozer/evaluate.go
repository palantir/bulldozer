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

package bulldozer

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"

	"github.com/palantir/bulldozer/pull"
)

// IsPRIgnored returns true if the PR is identified as ignored,
// false otherwise. Additionally, a description of the reason will be returned.
func IsPRIgnored(ctx context.Context, pullCtx pull.Context, config Signals) (bool, string, error) {
	matches, reason, err := config.Matches(ctx, pullCtx, "ignored")
	if err != nil {
		// ignore must always fail closed (matches on error)
		matches = true
	}
	return matches, reason, err
}

// IsPRTriggered returns true if the PR is identified as triggered,
// false otherwise. Additionally, a description of the reason will be returned.
func IsPRTriggered(ctx context.Context, pullCtx pull.Context, config Signals) (bool, string, error) {
	matches, reason, err := config.Matches(ctx, pullCtx, "triggered")
	if err != nil {
		// trigger must always fail closed (no match on error)
		return false, reason, err
	}
	return matches, reason, err
}

// statusSetDifference returns all statuses in required that are not in actual,
// accouting for special behavior in GitHub.
func statusSetDifference(required, actual []string) []string {
	// GitHub apparently implements special behavior with required statuses for
	// Travis CI for what I assume are legacy reasons. If travisStatusBase is
	// required, both travisStatusPush and travisStatusPR inherit the required
	// flag. Futher, travisStatusBase is the _only_ status of the three that
	// can be selected as required in the UI; the other two are hidden from
	// users. See issue #190 for more details.
	//
	// To account for this, pretend that travisStatusBase appears as a status
	// if either of the others appear in the actual list.
	const (
		travisStatusBase = "continuous-integration/travis-ci"
		travisStatusPush = "continuous-integration/travis-ci/push"
		travisStatusPR   = "continuous-integration/travis-ci/pr"
	)

	actualSet := make(map[string]struct{})
	for _, s := range actual {
		if s == travisStatusPush || s == travisStatusPR {
			actualSet[travisStatusBase] = struct{}{}
		}
		actualSet[s] = struct{}{}
	}

	seen := make(map[string]struct{})
	var result []string
	for _, s := range required {
		if _, ok := actualSet[s]; !ok {
			if _, alreadySeen := seen[s]; !alreadySeen {
				result = append(result, s)
				seen[s] = struct{}{}
			}
		}
	}
	return result
}

// ShouldMergePR determines if a given PullRequest should be merged, given the mergeConfig
func ShouldMergePR(ctx context.Context, pullCtx pull.Context, mergeConfig MergeConfig) (bool, string, error) {
	logger := zerolog.Ctx(ctx)

	if mergeConfig.Ignore.Enabled() {
		ignored, reason, err := IsPRIgnored(ctx, pullCtx, mergeConfig.Ignore)
		if err != nil {
			return false, "", errors.Wrap(err, "failed to determine if pull request is ignored")
		}
		if ignored {
			msg := fmt.Sprintf("%s is deemed not mergeable because ignoring is enabled and %s", pullCtx.Locator(), reason)
			logger.Debug().Msg(msg)
			return false, msg, nil
		}
	} else {
		logger.Debug().Msg("ignoring is not enabled")
	}

	if mergeConfig.Trigger.Enabled() {
		triggered, reason, err := IsPRTriggered(ctx, pullCtx, mergeConfig.Trigger)
		if err != nil {
			return false, "", errors.Wrap(err, "failed to determine if pull request is triggered")
		}
		if !triggered {
			msg := fmt.Sprintf("%s is deemed not mergeable because triggering is enabled and no trigger signal detected", pullCtx.Locator())
			logger.Debug().Msg(msg)
			return false, msg, nil
		}

		logger.Debug().Msgf("%s is triggered because triggering is enabled and %s", pullCtx.Locator(), reason)
	} else {
		logger.Debug().Msg("triggering is not enabled")
	}

	requiredStatuses, err := pullCtx.RequiredStatuses(ctx)
	if err != nil {
		return false, "", errors.Wrap(err, "failed to determine required Github status checks")
	}
	requiredStatuses = append(requiredStatuses, mergeConfig.RequiredStatuses...)

	successStatuses, err := pullCtx.CurrentSuccessStatuses(ctx)
	if err != nil {
		return false, "", errors.Wrap(err, "failed to determine currently successful status checks")
	}

	unsatisfiedStatuses := statusSetDifference(requiredStatuses, successStatuses)
	if len(unsatisfiedStatuses) > 0 {
		msg := fmt.Sprintf("%s is deemed not mergeable because of unfulfilled status checks: [%s]", pullCtx.Locator(), strings.Join(unsatisfiedStatuses, ","))
		logger.Debug().Msgf(msg)
		return false, msg, nil
	}

	// Ignore required reviews and try a merge (which may fail with a 4XX).
	return true, "", nil
}
