# bulldozer

`bulldozer` is a [GitHub App](https://developer.github.com/apps/) that auto-merges
PRs when all status checks are green and the PR is reviewed

## Configuration

By default, the behavior of the bot is configured by a `.bulldozer.yml` file at
the root of the repository. The file name and location are configurable when
running your own instance of the server.

- If the file does not exist, `bulldozer` will attempt to read from the configuration_v0_path, which is
configurable, though the default is `.bulldozer.yml` If `bulldozer` cannot find either configuration file,
it will take no action. This means it is safe to enable `bulldozer` on all repositories in an organization.

- The `.bulldozer.yml` file is read from most recent commit on the target branch
of each pull request.

## Behaviour

When `bulldozer` is enabled on a repo, it will merge all PRs as the `bulldozer[bot]`
committer. Behaviour is configured by a file in each repository.

We recommend using the following configuration, which can be copied into your configuration file.

```yaml
version: 1

# "merge" defines how to merge PRs into the target
merge:

  # "whitelist" defines how to select PRs to evaluate and merge
  whitelist:

    # "labels" is a list of labels that must be matched to whitelist a PR for merging
    labels: ["merge when ready"]

    # "comment_substrings" matches on substrings in comments
    comment_substrings: ["==MERGE_WHEN_READY=="]

  # "blacklist" defines how to exclude PRs from evaluation and merging
  blacklist:

    # similar as above, "labels" defines a list of labels. In this case, matched labels cause exclusion.
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


### Caveats and Notes

If both `blacklist` and `whitelist` are specified, `bulldozer` will attempt to match on both. 
In cases where both match, `blacklist` will take precedence.

The `merge_method` specifies the strategy that will be used to merge. Possible choices
are `merge`, `squash`, and `rebase`. Specifying `squash` will allow for a further
set of `squash_strategy` options, `pull_request_body`, `summarize_commits` and
`empty_body` that will constitute the body of the merge commit message. 

## Deployment

`bulldozer` is easy to deploy in your own environment as it has no dependencies
other than GitHub. It is also safe to run multiple instances of the server,
making it a good fit for container schedulers like Nomad or Kubernetes.

We provide both a Docker container and a binary distribution of the server. A
sample configuration file is provided at `var/conf/bulldozer.yml`. We
recommend deploying the application behind a reverse proxy or load balancer
that terminates TLS connections.

### GitHub App Configuration

`bulldozer` requires the following permissions as a GitHub app:

* Repository Admin - read-only
* Repository Contents - read & write
* Issues - read-only
* Repository metadata - read-only
* Pull requests - read-only
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

`bulldozer` uses [go-baseapp](https://github.com/palantir/go-baseapp) and
[go-githubapp](https://github.com/palantir/go-githubapp), both of which emit
standard metrics and structured log keys. Please see those projects for
details.

### Example Files
Example `bulldozer` files can be found in [`config/examples`](https://github.com/palantir/bulldozer/tree/develop/config/examples)

## Contributing

Contributions and issues are welcome. For new features or large contributions,
we prefer discussing the proposed change on a GitHub issue prior to a PR.

## License

This library is made available under the [Apache 2.0 License](http://www.apache.org/licenses/LICENSE-2.0).
