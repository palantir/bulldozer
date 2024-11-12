// Copyright 2024 Palantir Technologies, Inc.
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

// pull_test.go

package pull

import (
	"context"
	"testing"

	"github.com/google/go-github/v66/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockGitHubClient struct {
	mock.Mock
}

func (m *MockGitHubClient) ListPullRequestsWithCommit(ctx context.Context, owner, repo, sha string, opts *github.ListOptions) ([]*github.PullRequest, *github.Response, error) {
	args := m.Called(ctx, owner, repo, sha, opts)
	return args.Get(0).([]*github.PullRequest), args.Get(1).(*github.Response), args.Error(2)
}

func (m *MockGitHubClient) List(ctx context.Context, owner, repo string, opts *github.PullRequestListOptions) ([]*github.PullRequest, *github.Response, error) {
	args := m.Called(ctx, owner, repo, opts)
	return args.Get(0).([]*github.PullRequest), args.Get(1).(*github.Response), args.Error(2)
}

func TestGetOpenPullRequestsForSHA(t *testing.T) {
	mockClient := new(MockGitHubClient)
	ctx := context.Background()
	owner := "owner"
	repo := "repo"
	sha := "sha"

	pr := &github.PullRequest{
		State: github.String("open"),
		Head:  &github.PullRequestBranch{SHA: github.String(sha)},
	}

	mockClient.On("ListPullRequestsWithCommit", ctx, owner, repo, sha, mock.Anything).Return([]*github.PullRequest{pr}, &github.Response{NextPage: 0}, nil)

	prs, err := getOpenPullRequestsForSHA(ctx, mockClient, owner, repo, sha)
	assert.NoError(t, err)
	assert.Len(t, prs, 1)
	assert.Equal(t, sha, prs[0].GetHead().GetSHA())

	mockClient.AssertExpectations(t)
}

func TestListOpenPullRequestsForSHA(t *testing.T) {
	mockClient := new(MockGitHubClient)
	ctx := context.Background()
	owner := "owner"
	repo := "repo"
	sha := "sha"

	pr := &github.PullRequest{
		State: github.String("open"),
		Head:  &github.PullRequestBranch{SHA: github.String(sha)},
	}

	mockClient.On("List", ctx, owner, repo, mock.Anything).Return([]*github.PullRequest{pr}, &github.Response{NextPage: 0}, nil)

	prs, err := ListAllOpenPullRequestsFilteredBySHA(ctx, mockClient, owner, repo, sha)
	assert.NoError(t, err)
	assert.Len(t, prs, 1)
	assert.Equal(t, sha, prs[0].GetHead().GetSHA())

	mockClient.AssertExpectations(t)
}

func TestGetAllPossibleOpenPullRequestsForSHA_FirstMethodReturnsResults(t *testing.T) {
	mockClient := new(MockGitHubClient)
	ctx := context.Background()
	owner := "owner"
	repo := "repo"
	sha := "sha"

	pr := &github.PullRequest{
		State: github.String("open"),
		Head:  &github.PullRequestBranch{SHA: github.String(sha)},
	}

	// Mock the first method to return a valid pull request.
	mockClient.On("ListPullRequestsWithCommit", ctx, owner, repo, sha, mock.Anything).Return([]*github.PullRequest{pr}, &github.Response{NextPage: 0}, nil).Once()
	// Mock the second method to not be called.
	mockClient.On("List", ctx, owner, repo, mock.Anything).Return(nil, nil, nil).Maybe()

	prs, err := GetAllPossibleOpenPullRequestsForSHA(ctx, mockClient, owner, repo, sha)
	assert.NoError(t, err)
	assert.Len(t, prs, 1)
	assert.Equal(t, sha, prs[0].GetHead().GetSHA())

	mockClient.AssertExpectations(t)
}

func TestGetAllPossibleOpenPullRequestsForSHA_SecondMethodReturnsResults(t *testing.T) {
	mockClient := new(MockGitHubClient)
	ctx := context.Background()
	owner := "owner"
	repo := "repo"
	sha := "sha"

	pr := &github.PullRequest{
		State: github.String("open"),
		Head:  &github.PullRequestBranch{SHA: github.String(sha)},
	}

	// Mock the first method to return no results.
	mockClient.On("ListPullRequestsWithCommit", ctx, owner, repo, sha, mock.Anything).Return([]*github.PullRequest{}, &github.Response{NextPage: 0}, nil).Once()
	// Mock the second method to return a valid pull request.
	mockClient.On("List", ctx, owner, repo, mock.Anything).Return([]*github.PullRequest{pr}, &github.Response{NextPage: 0}, nil).Once()

	prs, err := GetAllPossibleOpenPullRequestsForSHA(ctx, mockClient, owner, repo, sha)
	assert.NoError(t, err)
	assert.Len(t, prs, 1)
	assert.Equal(t, sha, prs[0].GetHead().GetSHA())

	mockClient.AssertExpectations(t)
}

func TestGetAllPossibleOpenPullRequestsForSHA_NoResults(t *testing.T) {
	mockClient := new(MockGitHubClient)
	ctx := context.Background()
	owner := "owner"
	repo := "repo"
	sha := "sha"

	// Mock both methods to return no results.
	mockClient.On("ListPullRequestsWithCommit", ctx, owner, repo, sha, mock.Anything).Return([]*github.PullRequest{}, &github.Response{NextPage: 0}, nil).Once()
	mockClient.On("List", ctx, owner, repo, mock.Anything).Return([]*github.PullRequest{}, &github.Response{NextPage: 0}, nil).Once()

	prs, err := GetAllPossibleOpenPullRequestsForSHA(ctx, mockClient, owner, repo, sha)
	assert.NoError(t, err)
	assert.Len(t, prs, 0)

	mockClient.AssertExpectations(t)
}

func TestGetAllPossibleOpenPullRequestsForSHA_Errors(t *testing.T) {
	mockClient := new(MockGitHubClient)
	ctx := context.Background()
	owner := "owner"
	repo := "repo"
	sha := "sha"

	// Mock the first method to return an error.
	mockClient.On("ListPullRequestsWithCommit", ctx, owner, repo, sha, mock.Anything).Return([]*github.PullRequest{}, &github.Response{}, assert.AnError).Once()
	// Mock the second method to not be called.
	mockClient.On("List", ctx, owner, repo, mock.Anything).Return(nil, nil, nil).Maybe()

	prs, err := GetAllPossibleOpenPullRequestsForSHA(ctx, mockClient, owner, repo, sha)
	assert.Error(t, err)
	assert.Nil(t, prs)

	mockClient.AssertExpectations(t)
}

func TestListOpenPullRequestsForRef(t *testing.T) {
	mockClient := new(MockGitHubClient)
	ctx := context.Background()
	owner := "owner"
	repo := "repo"
	ref := "refs/heads/main"

	pr := &github.PullRequest{
		State: github.String("open"),
		Base:  &github.PullRequestBranch{Ref: github.String("main")},
	}

	mockClient.On("List", ctx, owner, repo, mock.Anything).Return([]*github.PullRequest{pr}, &github.Response{NextPage: 0}, nil)

	prs, err := GetAllOpenPullRequestsForRef(ctx, mockClient, owner, repo, ref)
	assert.NoError(t, err)
	assert.Len(t, prs, 1)
	assert.Equal(t, "main", prs[0].GetBase().GetRef())

	mockClient.AssertExpectations(t)
}
