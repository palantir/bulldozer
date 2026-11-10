// Copyright 2026 Palantir Technologies, Inc.
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
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-github/v93/github"
	"github.com/palantir/go-githubapp/githubapp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type staticClientCreator struct {
	githubapp.ClientCreator
	client *github.Client
}

func (cc *staticClientCreator) NewInstallationClient(installationID int64) (*github.Client, error) {
	return cc.client, nil
}

func TestIssueCommentOnlyLoadsPullRequests(t *testing.T) {
	tests := map[string]struct {
		Issue            string
		ExpectedRequests []string
	}{
		"issue": {
			Issue: `{"number": 1}`,
		},
		"pullRequest": {
			Issue:            `{"number": 1, "pull_request": {"url": "https://api.github.com/repos/owner/repo/pulls/1"}}`,
			ExpectedRequests: []string{"GET /api/v3/repos/owner/repo/pulls/1"},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			// The server has no pull requests, so loading one fails the handler
			var requests []string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests = append(requests, r.Method+" "+r.URL.Path)
				http.NotFound(w, r)
			}))
			defer srv.Close()

			client, err := github.NewClient(github.WithEnterpriseURLs(srv.URL, srv.URL))
			require.NoError(t, err)

			h := &IssueComment{Base: Base{ClientCreator: &staticClientCreator{client: client}}}
			payload := fmt.Appendf(nil, `{
				"action": "created",
				"issue": %s,
				"repository": {"name": "repo", "owner": {"login": "owner"}},
				"installation": {"id": 1}
			}`, test.Issue)

			err = h.Handle(context.Background(), "issue_comment", "delivery", payload)
			if len(test.ExpectedRequests) == 0 {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
			assert.Equal(t, test.ExpectedRequests, requests)
		})
	}
}
