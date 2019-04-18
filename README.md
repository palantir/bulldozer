# bulldozer

[![Download](https://api.bintray.com/packages/palantir/releases/bulldozer/images/download.svg)](https://bintray.com/palantir/releases/bulldozer/_latestVersion) [![Docker Pulls](https://img.shields.io/docker/pulls/palantirtechnologies/bulldozer.svg)](https://hub.docker.com/r/palantirtechnologies/bulldozer/)

`bulldozer` is a [GitHub App](https://developer.github.com/apps/) that
automatically merges pull requests (PRs) when (and only when) all required
status checks are successful and required reviews are provided.

Additionally, `bulldozer` can:

- Only merge pull requests that match a whitelist condition, like having a
  specific label or comment
- Ignore pull requests that match a blacklist condition, like having a specific
  label or comment
- Automatically keep pull request branches up-to-date by merging in the target
  branch
- Wait for additional status checks that are not required by GitHub

Bulldozer might be useful if you:

- Have CI builds that take longer than the normal review process. It will merge
  reviewed PRs as soon as the tests pass so you don't have to watch the pull
  request or remember to merge it later.
- Combine it with [policy-bot](https://github.com/palantir/policy-bot) to
  automatically merge certain types of pre-approved or automated changes.
- Want to give contributors more control over when they can merge PRs without
  granting them write access to the repository.
- Have a lot of active development that makes it difficult to merge a pull
  request while it is up-to-date with the target branch.

## Contents

* [Behavior](#behavior)
* [Configuration](#configuration)
  + [bulldozer.yml Specification](#bulldozeryml-specification)
* [FAQ](#faq)
* [Deployment](#deployment)
* [Development](#development)
* [Contributing](#contributing)
* [License](#license)

## Behavior

`bulldozer` will only merge pull requests that GitHub allows non-admin
collaborators to merge. This means that all branch protection settings,
including required status checks and required reviews, are respected. It also
means that you _must_ enable branch protection to prevent `bulldozer` from
immediately merging every pull request.

Only pull requests matching the whitelist conditions (or _not_ matching the
blacklist conditions) are considered for merging. `bulldozer` is event-driven,
which means it will usually merge a pull request within a few seconds of the
pull request satisfying all preconditions.

## Configuration

The behavior of the bot is configured by a `.bulldozer.yml` file at the root of
the repository. The file name and location are configurable when running your
own instance of the server.

The `.bulldozer.yml` file is read from the most recent commit on the target
branch of each pull request. If `bulldozer` cannot find a configuration file,
it will take no action. This means it is safe to enable the `bulldozer` on all.
repositories in an organization.

### bulldozer.yml Specification

The `.bulldozer.yml` file supports the following keys.

```yaml
# "version" is the configuration version, currently "1".
version: 1

# "merge" defines how and when pull requests are merged. If the section is
# missing, bulldozer will consider all pull requests and use default settings.
merge:
  # "whitelist" defines the set of pull requests considered by bulldozer. If
  # the section is missing, bulldozer considers all pull requests not excluded
  # by the blacklist.
  whitelist:
    # Pull requests with any of these labels (case-insensitive) are added to
    # the whitelist.
    labels: ["merge when ready"]

    # Pull requests where the body or any comment contains any of these
    # substrings are added to the whitelist.
    comment_substrings: ["==MERGE_WHEN_READY=="]

    # Pull requests where any comment matches one of these exact strings are
    # added to the whitelist.
    comments: ["Please merge this pull request!"]

    # Pull requests where the body contains any of these substrings are added
    # to the whitelist.
    pr_body_substrings: ["==MERGE_WHEN_READY=="]

    # Pull requests targeting any of these branches are added to the whitelist.
    branches: ["develop"]

  # "blacklist" defines the set of pull request ignored by bulldozer. If the
  # section is missing, bulldozer considers all pull requests. It takes the
  # same keys as the "whitelist" section.
  blacklist:
    labels: ["do not merge"]
    comment_substrings: ["==DO_NOT_MERGE=="]

  # "method" defines the merge method. The available options are "merge",
  # "rebase", and "squash".
  method: squash

  # Allows the merge method that is used when auto-merging a PR to be different based on the
  # target branch. The keys of the hash are the target branch name, and the values are the merge method that
  # will be used for PRs targeting that branch. The valid values are the same as for the "method" key.
  # Note: If the target branch does not match any of the specified keys, the "method" key is used instead.
  branch_method:
    develop: squash
    master: merge

  # "options" defines additional options for the individual merge methods.
  options:
    # "squash" options are only used when the merge method is "squash"
    squash:
      # "title" defines how the title of the commit message is created when
      # generating a squash commit. The options are "pull_request_title",
      # "first_commit_title", and "github_default_title". The default is
      # "pull_request_title".
      title: "pull_request_title"

      # "body" defines how the body of the commit message is created when
      # generating a squash commit. The options are "pull_request_body",
      # "summarize_commits", and "empty_body". The default is "empty_body".
      body: "empty_body"

      # If "body" is "pull_request_body", then the commit message will be the
      # part of the pull request body surrounded by "message_delimiter"
      # strings. This is disabled (empty string) by default.
      message_delimiter: ==COMMIT_MSG==

  # "required_status" is a list of additional status contexts that must pass
  # before bulldozer can merge a pull request. This is useful if you want to
  # require extra testing for automated merges, but not for manual merges.
  required_statuses:
    - "ci/circleci: ete-tests"

  # If true, bulldozer will delete branches after their pull requests merge.
  delete_after_merge: true

# "update" defines how and when to update pull request branches. Unlike with
# merges, if this section is missing, bulldozer will not update any pull requests.
update:
  # "whitelist" defines the set of pull requests that should be updated by
  # bulldozer. It accepts the same keys as the whitelist in the "merge" block.
  whitelist:
    labels: ["WIP", "Update Me"]

  # "blacklist" defines the set of pull requests that should not be updated by
  # bulldozer. It accepts the same keys as the blacklist in the "merge" block.
  blacklist:
    labels: ["Do Not Update"]
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

You can also define part of pull request body to pick as a commit message when
`body` is `pull_request_body`.

```yaml
merge:
  method: squash
  options:
    squash:
      body: pull_request_body
      message_delimiter: ==COMMIT_MSG==
```

Anything that's contained between two `==COMMIT_MSG==` strings will become the
commit message instead of whole pull request body.

#### What if I don't want to put config files into each repo?

You can add default repository configuration in your bulldozer config file.

It will be used only when your repo config file does not exist.

```yaml
options:
  default_repository_config:
    blacklist:
      labels: ["do not merge"] # or any other available config.
```

#### Bulldozer isn't merging my commit or updating my branch when it should, what could be happening?

Bulldozer will attempt to merge a branch whenever it passes the whitelist/blacklist
criteria. GitHub may prevent it from merging a branch in certain conditions, some of
which are to be expected, and others that may be caused by mis-configuring Bulldozer.

* Required status checks have not passed
* Review requirements are not satisfied
* The merge strategy configured in `.bulldozer.yml` is not allowed by your
  repository settings
* Branch protection rules are preventing `bulldozer[bot]` from [pushing to the
  branch][push restrictions]. Unfortunately, GitHub apps cannot be added to
  the list at this time, but there is [a workaround][] if you are running your
  own Bulldozer server.

[push restrictions]: https://help.github.com/articles/about-branch-restrictions/
[a workaround]: #can-bulldozer-work-with-push-restrictions-on-branches

#### Can Bulldozer work with push restrictions on branches?

As mentioned above, GitHub Apps cannot be added to the list of users associated
with [push restrictions][]. To work around this, you can:

1. Use another app like [policy-bot](https://github.com/palantir/policy-bot) to
   implement _approval_ restrictions as required status checks instead of using
   push restrictions. This effectively limits who can push to a branch by
   requiring changes to go through the pull request process and be approved.

2. Configure Bulldozer to use a personal access token for a regular user to
   perform merges in this case. The token must have the `repo` scope and the
   user must be allowed to push to the branch. In the server configuration
   file, set:

   ```yaml
   options:
     push_restriction_user_token: <token-value>
   ```

   The token is _only_ used if the target branch has push restrictions enabled.
   All other merges are performed as the normal GitHub App user.

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

Bulldozer requires the following permissions as a GitHub app:

| Permission | Access | Reason |
| ---------- | ------ | ------ |
| Repository administration | Read-only | Determine required status checks |
| Checks | Read-only | Read checks for ref |
| Repository contents | Read & write | Read configuration, perform merges |
| Issues | Read & write | Read comments, close linked issues |
| Repository metadata | Read-only | Basic repository data |
| Pull requests | Read & write | Merge and close pull requests |
| Commit status | Read-only | Evaluate pull request status |

It should be subscribed to the following events:

* Check run
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

## Development

To develop `bulldozer`, you will need a [Go installation](https://golang.org/doc/install).

**Run style checks and tests**

    ./godelw verify

**Running the server locally**

    # copy and edit the server config
    cp config/bulldozer.example.yml config/bulldozer.yml

    ./godelw run bulldozer server

- `config/bulldozer.yml` is used as the default configuration file
- The server is available at `http://localhost:8080/`

**Running the server via Docker**

    # copy and edit the server config
    cp config/bulldozer.example.yml config/bulldozer.yml

    # build the docker image
    ./godelw docker build --verbose

    docker run --rm -v "$(pwd)/config:/secrets/" -p 8080:8080 palantirtechnologies/bulldozer:latest

- This mounts the `config` directory (which should contain the `bulldozer.yml` configuration file) at the expected location
- The server is available at `http://localhost:8080/`

## Contributing

Contributions and issues are welcome. For new features or large contributions,
we prefer discussing the proposed change on a GitHub issue prior to a PR.

## License

This application is made available under the [Apache 2.0 License](http://www.apache.org/licenses/LICENSE-2.0).
