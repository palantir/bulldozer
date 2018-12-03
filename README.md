# bulldozer

[![Download](https://api.bintray.com/packages/palantir/releases/bulldozer/images/download.svg)](https://bintray.com/palantir/releases/bulldozer/_latestVersion) [![Docker Pulls](https://img.shields.io/docker/pulls/palantirtechnologies/bulldozer.svg)](https://hub.docker.com/r/palantirtechnologies/bulldozer/)

bulldozer is a [GitHub App](https://developer.github.com/apps/) that auto-merges
pull requests (PRs) when all status checks are green and the PR is reviewed.

## Configuration

By default, the behavior of the bot is configured by a `.bulldozer.yml` file at
the root of the repository. The file name and location are configurable when
running your own instance of the server. The `.bulldozer.yml` file is read from
most recent commit on the target branch of each pull request. If bulldozer cannot
find a configuration file, it will take no action. This means it is safe to enable
the bulldozer Github App on all repositories in an organization.

## Behaviour

When bulldozer is enabled on a repo, it will merge all PRs as the `bulldozer[bot]`
committer. Behaviour is configured by a file in each repository.

We recommend using the following configuration, which can be copied into your configuration file.

```yaml
version: 1

# "merge" defines how to merge PRs into the target
merge:

  # "whitelist" defines how to select PRs to evaluate and merge
  whitelist:

    # "labels" is a list of labels that must be matched to whitelist a PR for merging (case-insensitive)
    labels: ["merge when ready"]

    # "comment_substrings" matches on substrings in comments or the pull request body
    comment_substrings: ["==MERGE_WHEN_READY=="]

    # "pr_body_substrings" matches on substrings in the PR body
    pr_body_substrings: ["==MERGE_WHEN_READY=="]

  # "blacklist" defines how to exclude PRs from evaluation and merging
  blacklist:

    # similar as above, "labels" defines a list of labels. In this case, matched labels cause exclusion. (case-insensitive)
    labels: ["do not merge"]

    # "comment_substrings" matches substrings in comments. In this case, matched substrings cause exclusion.
    comment_substrings: ["==DO_NOT_MERGE=="]

  # "method" defines how to merge in changes. Available options are "merge", "rebase" and "squash"
  method: squash

  # "options" is used in conjunction with "method", and defines additional merging options for each type.
  options:

    # "squash" is used when the "method" above is set to "squash"
    squash:

      # "body" is a an option for handling the merge body. available options are "summarize_commits", "pull_request_body", and "empty_body"
      body: summarize_commits

  # "delete_after_merge" is a bool that will cause merged PRs to be deleted once they are successfully merged
  delete_after_merge: true

# "update" defines how to keep open PRs up to date
update:

  # The "whitelist" and "blacklist" options here operate the same as described for the `merge` block.
  whitelist:
    labels: ["WIP", "Update Me"]
```

## FAQ

#### Can I specify both a `blacklist` and `whitelist`?
Yes. If both `blacklist` and `whitelist` are specified, bulldozer will attempt to match
on both. In cases where both match, `blacklist` will take precedence.

#### Can I specify the body of the commit when using the `squash` strategy?
Yes. When the merge strategy is `squash`, you can set additional options under the
`options.squash` property, including how to render the commit body.

```yaml
merge:
  method: squash
  options:
    squash:
      body: summarize_commits # or `pull_request_body`, `empty_body`
```

#### What if I don't want to put config files into each repo?
You can enable global fallback configuration in your bulldozer config file.

Global configuration will be used only when your repo config file not exists.

```yaml
options:
  enable_global_config: true

global:
  blacklist:
    labels: ["do not merge"] # or any other available config.
```

#### Bulldozer isn't merging my commit or updating my branch when it should, what could be happening?
Bulldozer will attempt to merge a branch whenever it passes the whitelist/blacklist
criteria. GitHub may prevent it from merging a branch in certain conditions, some of
which are to be expected, and others that may be caused by mis-configuring Bulldozer.

* Required status checks have not passed
* Review requirements are not satisfied
* The merge strategy configured in `.bulldozer.yml` is not allowed by your repository settings
* Branch protection rules are preventing `bulldozer [bot]` from [pushing to the branch](https://help.github.com/articles/about-branch-restrictions/).
  Unfortunately GitHub apps cannot be added to the list at this time.

## Deployment

bulldozer is easy to deploy in your own environment as it has no dependencies
other than GitHub. It is also safe to run multiple instances of the server,
making it a good fit for container schedulers like Nomad or Kubernetes.

We provide both a Docker container and a binary distribution of the server:

- Binaries: https://bintray.com/palantir/releases/bulldozer
- Docker Images: https://hub.docker.com/r/palantirtechnologies/bulldozer/

A sample configuration file is provided at `config/bulldozer.example.yml`. We
recommend deploying the application behind a reverse proxy or load balancer
that terminates TLS connections.

### GitHub App Configuration

Webhook URL:

* `http(s)://your.domain.com/api/github/hook`

bulldozer requires the following permissions as a GitHub app:

* Repository Admin - read-only
* Repository Contents - read & write
* Issues - read-only
* Repository metadata - read-only
* Pull requests - read & write
* Commit status - read-only

It should be subscribed to the following events:

* Commit comment
* Pull request
* Status
* Push
* Issue comment
* Pull request review
* Pull request review comment

### Operations

bulldozer uses [go-baseapp](https://github.com/palantir/go-baseapp) and
[go-githubapp](https://github.com/palantir/go-githubapp), both of which emit
standard metrics and structured log keys. Please see those projects for
details.

### Example Files

Example `.bulldozer.yml` files can be found in [`config/examples`](https://github.com/palantir/bulldozer/tree/develop/config/examples)

### Migrating: Version 0.4.X to 1.X

The server configuration for bulldozer allows you to specify `configuration_v0_path`, which is a list of paths
to check for `0.4.X` style bulldozer configuration. When a `1.X` style configuration file does not appear
at the configured path, bulldozer will attempt to read from the paths configured by `configuration_v0_path`,
converting the legacy configuration into an equivalent `v1` configuration internally. 

The upgrade process is therefore to deploy the latest version of bulldozer with both `configuration_path` and
`configuration_v0_path` configured, and to enable the bulldozer GitHub App on all organizations where it was
previously installed.

## Contributing

Contributions and issues are welcome. For new features or large contributions,
we prefer discussing the proposed change on a GitHub issue prior to a PR.

## License

This application is made available under the [Apache 2.0 License](http://www.apache.org/licenses/LICENSE-2.0).
