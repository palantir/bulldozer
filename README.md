# Bulldozer

The `Bulldozer` is a bot that auto-merges PRs when all status checks are green and the PR is reviewed.
It uses [GitHub status checks](https://developer.github.com/v3/repos/statuses/) and [GitHub reviews](https://developer.github.com/v3/pulls/reviews/) to do this.


## Running the server

* The server expects a config file name to be passed-in with the `-c` parameter (see `./bulldozer --help`).  This config file should look like this.

```yml
rest:
  address: 0.0.0.0
  port: 80

logging:
  level: info

database:
  name: dbName
  host: dbHost
  username: dbUser
  password: dbPassword
  sslmode: require

github:
  address: https://github.com/
  apiURL: https://api.github.com/
  callbackURL: https://bulldozer.dev/login
  clientID: ???
  clientSecret: ???
  webhookURL: https://bulldozer.dev/api/github/hook
  webhookSecret: ???

assetDir: ???
```

You can get the `clientID` and `clientSecret` after you create an OAuth application. More details on this [here](https://developer.github.com/apps/building-integrations/setting-up-and-registering-oauth-apps/). The `assetsDir` config option should point to where you saved the web assets that `Bulldozer` serves. After building the project they can be found in the `client/build` folder.

## Behavior

When bulldozer is enabled on a repo, it will merge all PRs under the user which enabled it in the first place (enabled will be the commiter).

Bulldozer has the following behavior:

### there exists no `.bulldozer.yml` file in the destination branch of a PR
- Bulldozer will ignore the PR and will not act on it

### there exists a `.bulldozer.yml` file in the destination branch of a PR
The format of the file is:

```yaml
mode: whitelist/blacklist/pr_body
strategy: merge/squash/rebase
deleteAfterMerge: true/false
```

The `mode` block specifies in which mode Bulldozer should operate. Available options:
   - `whitelist` it will act on PRs that have a GitHub label with the text `MERGE WHEN READY`.
   - `blacklist` it will not act on PRs that have a GitHub label with the text `DO NOT MERGE` or `WIP`.
   - `pr_body` it will act on PRs that have in their body the text `==MERGE_WHEN_READY==`. This is useful for PRs from forks because usually people that fork the repo do not have write permissions on the source to attach the needed labels.

The `strategy` field allows you to tune the merge strategy. Available options:
  - `merge`: it will create a merge commit.
  - `squash`: it will squash all the commits into one and will apply it on the target branch. With this merge strategy the user can specify the commit message ahead of time by putting a block like the following `==COMMIT_MSG== sample commit message ==COMMIT_MSG==` in the PR body.
  - `rebase`: it will rebase all commits on top of the target branch.

If the desired merge method is not allowed by the repository one will be chosen by default in the following order `squash, merge, rebase` from the ones allowed.

The `deleteAfterMerge` block specifies if the source branch of the PR should be deleted after the merge is performed.

If the repository has branch protection active but does not require PRs to have at least one reviewer Bulldozer will just merge the PR unless the creator of the PR assigned a reviewer explicitly. In that case it will wait for at least one APPROVED review

If the repository has branch protection active and enforces code reviews on PRs Bulldozer will wait until all status checks are green and at least one APPROVED review exists on the PR before merging it

### Keeping PRs up to date with base branch
Bulldozer also has the option to update PR branches if the target branch has been updated. If the PR has a GitHub label with the text `UPDATE ME` then Bulldozer will keep this PR up to date with the base branch.

## Contributing
For general guidelines on contributing the Palantir products, see [this page](https://github.com/palantir/gradle-baseline/blob/develop/docs/best-practices/contributing/readme.md)
