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
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"

	"github.com/palantir/bulldozer/pull"
)

// IsPRBlocklisted returns true if the PR is identified as blocklisted,
// false otherwise. Additionally, a description of the reason will be returned.
func IsPRBlocklisted(ctx context.Context, pullCtx pull.Context, config Signals) (bool, string, error) {
	matches, reason, err := config.Matches(ctx, pullCtx, "blocklist")
	if err != nil {
		// blocklist must always fail closed (matches on error)
		matches = true
	}
	return matches, reason, err
}

// IsPRAllowlisted returns true if the PR is identified as allowlisted,
// false otherwise. Additionally, a description of the reason will be returned.
func IsPRAllowlisted(ctx context.Context, pullCtx pull.Context, config Signals) (bool, string, error) {
	matches, reason, err := config.Matches(ctx, pullCtx, "allowlist")
	if err != nil {
		// allowlist must always fail closed (no match on error)
		return false, reason, err
	}
	return matches, reason, err
}

// setDifference returns all elements in set1 that
// are not in set2.
func setDifference(set1, set2 []string) []string {
	m2 := make(map[string]struct{})
	for _, s2 := range set2 {
		m2[s2] = struct{}{}
	}

	seen := make(map[string]struct{})
	var result []string
	for _, s1 := range set1 {
		if _, ok := m2[s1]; !ok {
			if _, alreadySeen := seen[s1]; !alreadySeen {
				result = append(result, s1)
				seen[s1] = struct{}{}
			}
		}
	}
	return result
}

// ShouldMergePR TODO: may want to return a richer type than bool
func ShouldMergePR(ctx context.Context, pullCtx pull.Context, mergeConfig MergeConfig) (bool, error) {
	logger := zerolog.Ctx(ctx)

	if mergeConfig.Blocklist.Enabled() {
		blocklisted, reason, err := IsPRBlocklisted(ctx, pullCtx, mergeConfig.Blocklist)
		if err != nil {
			return false, errors.Wrap(err, "failed to determine if pull request is blocklisted")
		}
		if blocklisted {
			logger.Debug().Msgf("%s is deemed not mergeable because blocklisting is enabled and %s", pullCtx.Locator(), reason)
			return false, nil
		}
	} else {
		logger.Debug().Msg("blocklisting is not enabled")
	}

	if mergeConfig.Allowlist.Enabled() {
		allowlisted, reason, err := IsPRAllowlisted(ctx, pullCtx, mergeConfig.Allowlist)
		if err != nil {
			return false, errors.Wrap(err, "failed to determine if pull request is allowlisted")
		}
		if !allowlisted {
			logger.Debug().Msgf("%s is deemed not mergeable because allowlisting is enabled and no allowlist signal detected", pullCtx.Locator())
			return false, nil
		}

		logger.Debug().Msgf("%s is allowlisted because allowlisting is enabled and %s", pullCtx.Locator(), reason)
	} else {
		logger.Debug().Msg("allowlisting is not enabled")
	}

	requiredStatuses, err := pullCtx.RequiredStatuses(ctx)
	if err != nil {
		return false, errors.Wrap(err, "failed to determine required Github status checks")
	}
	requiredStatuses = append(requiredStatuses, mergeConfig.RequiredStatuses...)

	successStatuses, err := pullCtx.CurrentSuccessStatuses(ctx)
	if err != nil {
		return false, errors.Wrap(err, "failed to determine currently successful status checks")
	}

	unsatisfiedStatuses := setDifference(requiredStatuses, successStatuses)
	if len(unsatisfiedStatuses) > 0 {
		logger.Debug().Msgf("%s is deemed not mergeable because of unfulfilled status checks: [%s]", pullCtx.Locator(), strings.Join(unsatisfiedStatuses, ","))
		return false, nil
	}

	// Ignore required reviews and try a merge (which may fail with a 4XX).

	return true, nil
}
