package fail

fail

/*
This is a non-compiling file that has been added to explicitly ensure that CI fails.
It also contains the command that caused the failure and its output.
Remove this file if debugging locally.

go mod operation failed. This may mean that there are legitimate dependency issues with the "go.mod" definition in the repository and the updates performed by the gomod check. This branch can be cloned locally to debug the issue.

Command that caused error:
./godelw check compiles

Output:
Running compiles...
server/handler/base.go:77:81: cannot use tokenClient (variable of type *"github.com/google/go-github/v71/github".Client) as *"github.com/google/go-github/v70/github".Client value in argument to bulldozer.NewGitHubMerger
server/handler/check_run.go:46:57: cannot use &event (value of type *"github.com/google/go-github/v70/github".CheckRunEvent) as githubapp.InstallationSource value in argument to githubapp.GetInstallationIDFromEvent: *"github.com/google/go-github/v70/github".CheckRunEvent does not implement githubapp.InstallationSource (wrong type for method GetInstallation)
		have GetInstallation() *"github.com/google/go-github/v70/github".Installation
		want GetInstallation() *"github.com/google/go-github/v71/github".Installation
server/handler/check_run.go:48:67: cannot use repo (variable of type *"github.com/google/go-github/v70/github".Repository) as *"github.com/google/go-github/v71/github".Repository value in argument to githubapp.PrepareRepoContext
server/handler/check_run.go:64:61: cannot use client.PullRequests (variable of type *"github.com/google/go-github/v71/github".PullRequestsService) as pull.GitHubPullRequestClient value in argument to pull.ListAllOpenPullRequestsFilteredBySHA: *"github.com/google/go-github/v71/github".PullRequestsService does not implement pull.GitHubPullRequestClient (wrong type for method List)
		have List(context.Context, string, string, *"github.com/google/go-github/v71/github".PullRequestListOptions) ([]*"github.com/google/go-github/v71/github".PullRequest, *"github.com/google/go-github/v71/github".Response, error)
		want List(context.Context, string, string, *"github.com/google/go-github/v70/github".PullRequestListOptions) ([]*"github.com/google/go-github/v70/github".PullRequest, *"github.com/google/go-github/v70/github".Response, error)
server/handler/check_run.go:85:36: cannot use client (variable of type *"github.com/google/go-github/v71/github".Client) as *"github.com/google/go-github/v70/github".Client value in argument to pull.NewGithubContext
server/handler/check_run.go:85:44: cannot use fullPR (variable of type *"github.com/google/go-github/v71/github".PullRequest) as *"github.com/google/go-github/v70/github".PullRequest value in argument to pull.NewGithubContext
server/handler/check_run.go:87:42: cannot use client (variable of type *"github.com/google/go-github/v71/github".Client) as *"github.com/google/go-github/v70/github".Client value in argument to h.FetchConfigForPR
server/handler/check_run.go:87:50: cannot use fullPR (variable of type *"github.com/google/go-github/v71/github".PullRequest) as *"github.com/google/go-github/v70/github".PullRequest value in argument to h.FetchConfigForPR
server/handler/check_run.go:96:78: cannot use client (variable of type *"github.com/google/go-github/v71/github".Client) as *"github.com/google/go-github/v70/github".Client value in argument to h.UpdatePullRequest
server/handler/check_run.go:104:48: cannot use client (variable of type *"github.com/google/go-github/v71/github".Client) as *"github.com/google/go-github/v70/github".Client value in argument to h.ProcessPullRequest
server/handler/check_run.go:104:64: cannot use fullPR (variable of type *"github.com/google/go-github/v71/github".PullRequest) as *"github.com/google/go-github/v70/github".PullRequest value in argument to h.ProcessPullRequest
server/handler/fetcher.go:50:38: cannot use client (variable of type *"github.com/google/go-github/v70/github".Client) as *"github.com/google/go-github/v71/github".Client value in argument to cf.loader.LoadConfig
server/handler/issue_comment.go:45:57: cannot use &event (value of type *"github.com/google/go-github/v70/github".IssueCommentEvent) as githubapp.InstallationSource value in argument to githubapp.GetInstallationIDFromEvent: *"github.com/google/go-github/v70/github".IssueCommentEvent does not implement githubapp.InstallationSource (wrong type for method GetInstallation)
		have GetInstallation() *"github.com/google/go-github/v70/github".Installation
		want GetInstallation() *"github.com/google/go-github/v71/github".Installation
server/handler/issue_comment.go:46:65: cannot use repo (variable of type *"github.com/google/go-github/v70/github".Repository) as *"github.com/google/go-github/v71/github".Repository value in argument to githubapp.PreparePRContext
server/handler/issue_comment.go:59:35: cannot use client (variable of type *"github.com/google/go-github/v71/github".Client) as *"github.com/google/go-github/v70/github".Client value in argument to pull.NewGithubContext
server/handler/issue_comment.go:59:43: cannot use pr (variable of type *"github.com/google/go-github/v71/github".PullRequest) as *"github.com/google/go-github/v70/github".PullRequest value in argument to pull.NewGithubContext
server/handler/issue_comment.go:61:41: cannot use client (variable of type *"github.com/google/go-github/v71/github".Client) as *"github.com/google/go-github/v70/github".Client value in argument to h.FetchConfigForPR
server/handler/issue_comment.go:61:49: cannot use pr (variable of type *"github.com/google/go-github/v71/github".PullRequest) as *"github.com/google/go-github/v70/github".PullRequest value in argument to h.FetchConfigForPR
server/handler/issue_comment.go:65:47: cannot use client (variable of type *"github.com/google/go-github/v71/github".Client) as *"github.com/google/go-github/v70/github".Client value in argument to h.ProcessPullRequest
server/handler/issue_comment.go:65:63: cannot use pr (variable of type *"github.com/google/go-github/v71/github".PullRequest) as *"github.com/google/go-github/v70/github".PullRequest value in argument to h.ProcessPullRequest
server/handler/pull_request.go:45:57: cannot use &event (value of type *"github.com/google/go-github/v70/github".PullRequestEvent) as githubapp.InstallationSource value in argument to githubapp.GetInstallationIDFromEvent: *"github.com/google/go-github/v70/github".PullRequestEvent does not implement githubapp.InstallationSource (wrong type for method GetInstallation)
		have GetInstallation() *"github.com/google/go-github/v70/github".Installation
		want GetInstallation() *"github.com/google/go-github/v71/github".Installation
server/handler/pull_request.go:46:65: cannot use repo (variable of type *"github.com/google/go-github/v70/github".Repository) as *"github.com/google/go-github/v71/github".Repository value in argument to githubapp.PreparePRContext
server/handler/pull_request.go:64:35: cannot use client (variable of type *"github.com/google/go-github/v71/github".Client) as *"github.com/google/go-github/v70/github".Client value in argument to pull.NewGithubContext
server/handler/pull_request.go:64:43: cannot use pr (variable of type *"github.com/google/go-github/v71/github".PullRequest) as *"github.com/google/go-github/v70/github".PullRequest value in argument to pull.NewGithubContext
server/handler/pull_request.go:66:41: cannot use client (variable of type *"github.com/google/go-github/v71/github".Client) as *"github.com/google/go-github/v70/github".Client value in argument to h.FetchConfigForPR
server/handler/pull_request.go:66:49: cannot use pr (variable of type *"github.com/google/go-github/v71/github".PullRequest) as *"github.com/google/go-github/v70/github".PullRequest value in argument to h.FetchConfigForPR
server/handler/pull_request.go:76:78: cannot use client (variable of type *"github.com/google/go-github/v71/github".Client) as *"github.com/google/go-github/v70/github".Client value in argument to h.UpdatePullRequest
server/handler/pull_request.go:76:94: cannot use pr (variable of type *"github.com/google/go-github/v71/github".PullRequest) as *"github.com/google/go-github/v70/github".PullRequest value in argument to h.UpdatePullRequest
server/handler/pull_request.go:87:47: cannot use client (variable of type *"github.com/google/go-github/v71/github".Client) as *"github.com/google/go-github/v70/github".Client value in argument to h.ProcessPullRequest
server/handler/pull_request.go:87:63: cannot use pr (variable of type *"github.com/google/go-github/v71/github".PullRequest) as *"github.com/google/go-github/v70/github".PullRequest value in argument to h.ProcessPullRequest
server/handler/pull_request_review.go:45:57: cannot use &event (value of type *"github.com/google/go-github/v70/github".PullRequestReviewEvent) as githubapp.InstallationSource value in argument to githubapp.GetInstallationIDFromEvent: *"github.com/google/go-github/v70/github".PullRequestReviewEvent does not implement githubapp.InstallationSource (wrong type for method GetInstallation)
		have GetInstallation() *"github.com/google/go-github/v70/github".Installation
		want GetInstallation() *"github.com/google/go-github/v71/github".Installation
server/handler/pull_request_review.go:46:65: cannot use repo (variable of type *"github.com/google/go-github/v70/github".Repository) as *"github.com/google/go-github/v71/github".Repository value in argument to githubapp.PreparePRContext
server/handler/pull_request_review.go:59:35: cannot use client (variable of type *"github.com/google/go-github/v71/github".Client) as *"github.com/google/go-github/v70/github".Client value in argument to pull.NewGithubContext
server/handler/pull_request_review.go:59:43: cannot use pr (variable of type *"github.com/google/go-github/v71/github".PullRequest) as *"github.com/google/go-github/v70/github".PullRequest value in argument to pull.NewGithubContext
server/handler/pull_request_review.go:61:41: cannot use client (variable of type *"github.com/google/go-github/v71/github".Client) as *"github.com/google/go-github/v70/github".Client value in argument to h.FetchConfigForPR
server/handler/pull_request_review.go:61:49: cannot use pr (variable of type *"github.com/google/go-github/v71/github".PullRequest) as *"github.com/google/go-github/v70/github".PullRequest value in argument to h.FetchConfigForPR
server/handler/pull_request_review.go:65:47: cannot use client (variable of type *"github.com/google/go-github/v71/github".Client) as *"github.com/google/go-github/v70/github".Client value in argument to h.ProcessPullRequest
server/handler/pull_request_review.go:65:63: cannot use pr (variable of type *"github.com/google/go-github/v71/github".PullRequest) as *"github.com/google/go-github/v70/github".PullRequest value in argument to h.ProcessPullRequest
server/handler/push.go:44:57: cannot use &event (value of type *"github.com/google/go-github/v70/github".PushEvent) as githubapp.InstallationSource value in argument to githubapp.GetInstallationIDFromEvent: *"github.com/google/go-github/v70/github".PushEvent does not implement githubapp.InstallationSource (wrong type for method GetInstallation)
		have GetInstallation() *"github.com/google/go-github/v70/github".Installation
		want GetInstallation() *"github.com/google/go-github/v71/github".Installation
server/handler/push.go:55:67: cannot use ghRepo (variable of type *"github.com/google/go-github/v70/github".Repository) as *"github.com/google/go-github/v71/github".Repository value in argument to githubapp.PrepareRepoContext
server/handler/push.go:69:53: cannot use client.PullRequests (variable of type *"github.com/google/go-github/v71/github".PullRequestsService) as pull.GitHubPullRequestClient value in argument to pull.GetAllOpenPullRequestsForRef: *"github.com/google/go-github/v71/github".PullRequestsService does not implement pull.GitHubPullRequestClient (wrong type for method List)
		have List(context.Context, string, string, *"github.com/google/go-github/v71/github".PullRequestListOptions) ([]*"github.com/google/go-github/v71/github".PullRequest, *"github.com/google/go-github/v71/github".Response, error)
		want List(context.Context, string, string, *"github.com/google/go-github/v70/github".PullRequestListOptions) ([]*"github.com/google/go-github/v70/github".PullRequest, *"github.com/google/go-github/v70/github".Response, error)
server/handler/push.go:79:36: cannot use client (variable of type *"github.com/google/go-github/v71/github".Client) as *"github.com/google/go-github/v70/github".Client value in argument to h.FetchConfig
server/handler/push.go:88:36: cannot use client (variable of type *"github.com/google/go-github/v71/github".Client) as *"github.com/google/go-github/v70/github".Client value in argument to pull.NewGithubContext
server/handler/push.go:89:70: cannot use client (variable of type *"github.com/google/go-github/v71/github".Client) as *"github.com/google/go-github/v70/github".Client value in argument to h.UpdatePullRequest
server/handler/status.go:47:57: cannot use &event (value of type *"github.com/google/go-github/v70/github".StatusEvent) as githubapp.InstallationSource value in argument to githubapp.GetInstallationIDFromEvent: *"github.com/google/go-github/v70/github".StatusEvent does not implement githubapp.InstallationSource (wrong type for method GetInstallation)
		have GetInstallation() *"github.com/google/go-github/v70/github".Installation
		want GetInstallation() *"github.com/google/go-github/v71/github".Installation
server/handler/status.go:48:67: cannot use repo (variable of type *"github.com/google/go-github/v70/github".Repository) as *"github.com/google/go-github/v71/github".Repository value in argument to githubapp.PrepareRepoContext
server/handler/status.go:62:61: cannot use client.PullRequests (variable of type *"github.com/google/go-github/v71/github".PullRequestsService) as pull.GitHubPullRequestClient value in argument to pull.GetAllPossibleOpenPullRequestsForSHA: *"github.com/google/go-github/v71/github".PullRequestsService does not implement pull.GitHubPullRequestClient (wrong type for method List)
		have List(context.Context, string, string, *"github.com/google/go-github/v71/github".PullRequestListOptions) ([]*"github.com/google/go-github/v71/github".PullRequest, *"github.com/google/go-github/v71/github".Response, error)
		want List(context.Context, string, string, *"github.com/google/go-github/v70/github".PullRequestListOptions) ([]*"github.com/google/go-github/v70/github".PullRequest, *"github.com/google/go-github/v70/github".Response, error)
server/handler/status.go:73:36: cannot use client (variable of type *"github.com/google/go-github/v71/github".Client) as *"github.com/google/go-github/v70/github".Client value in argument to pull.NewGithubContext
server/handler/status.go:75:42: cannot use client (variable of type *"github.com/google/go-github/v71/github".Client) as *"github.com/google/go-github/v70/github".Client value in argument to h.FetchConfigForPR
server/handler/status.go:84:78: cannot use client (variable of type *"github.com/google/go-github/v71/github".Client) as *"github.com/google/go-github/v70/github".Client value in argument to h.UpdatePullRequest
server/handler/status.go:92:68: cannot use client (variable of type *"github.com/google/go-github/v71/github".Client) as *"github.com/google/go-github/v70/github".Client value in argument to h.ProcessPullRequest
Finished compiles
Check(s) produced output: [compiles]

*/
