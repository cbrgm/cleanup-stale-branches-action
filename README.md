# Cleanup Stale Branches Action

**Automatically delete abandoned / stale branches from a GitHub repository**

[![GitHub release](https://img.shields.io/github/release/cbrgm/cleanup-stale-branches-action.svg)](https://github.com/cbrgm/cleanup-stale-branches-action)
[![Go Report Card](https://goreportcard.com/badge/github.com/cbrgm/cleanup-stale-branches-action)](https://goreportcard.com/report/github.com/cbrgm/cleanup-stale-branches-action)
[![go-lint-test](https://github.com/cbrgm/cleanup-stale-branches-action/actions/workflows/go-lint-test.yml/badge.svg)](https://github.com/cbrgm/cleanup-stale-branches-action/actions/workflows/go-lint-test.yml)
[![go-binaries](https://github.com/cbrgm/cleanup-stale-branches-action/actions/workflows/go-binaries.yml/badge.svg)](https://github.com/cbrgm/cleanup-stale-branches-action/actions/workflows/go-binaries.yml)
[![container](https://github.com/cbrgm/cleanup-stale-branches-action/actions/workflows/container.yml/badge.svg)](https://github.com/cbrgm/cleanup-stale-branches-action/actions/workflows/container.yml)

## Stale / Abandoned Branch Criteria

This GitHub Action deems a branch as stale or abandoned based on the following criteria:

- 🚫 **Not Default Branch**: The branch is not the repository's default branch.
- 🛡️ **Not Protected**: The branch is not a protected branch.
- 📭 **No Open Pull Requests**: There are no open pull requests that originate from the branch.
- 🔀 **Not Base of an Open Pull Request**: The branch is not the base branch for any open pull requests.
- 🚫 **Not in Ignore List**: The branch is not included in the optional list of branches to ignore.
- ❌ **(No) Branch Prefix Match**: If specified, the branch name does (not) match one of the given prefixes.
- ⏰ **Latest Commit Age**: The branch's last commit is older than the specified number of days (e.g., no commits for 30 days).

Branches that meet all these criteria are considered as stale or abandoned and eligible for deletion.

## Inputs

- `token`: **Required** - GitHub token for authentication. Use GitHub secrets for security.
- `repository`: **Required** - The target GitHub repository in the format "owner/repo".
- `ignore-branches`: Optional - Comma-separated list of branches to ignore from deletion.
- `allowed-prefixes`: Optional - Comma-separated list of prefixes a branch must match to be considered for deletion.
- `ignored-prefixes`: Optional - Comma-separated list of prefixes a branch must NOT match to be considered for deletion.
- `last-commit-age-days`: Optional - Number of days since the last commit for a branch to be considered abandoned. Defaults to `30` days.
- `dry-run`: Optional - Perform a dry run without actually deleting branches. Defaults to `true`, meaning no branches will be deleted.
- `rate-limit`: Optional - Stop the action if it exceeds 95% of the GitHub API rate limit. Defaults to `true`, ensuring the action is halted before hitting the rate limit e.g. exiting with status code `0` instead of failing.

### Container Usage

This action can be executed independently from workflows within a container. To do so, use the following command:

```
podman run --rm -it ghcr.io/cbrgm/cleanup-stale-branches-action:v1 --help
```

### Workflow Usage

```yaml
name: Cleanup Stale Branches

on:
  workflow_dispatch:
  schedule:
    - cron: '0 0 * * *' # This schedule runs the workflow at midnight every day

jobs:
  cleanup-stale-branches:
    runs-on: ubuntu-latest
    steps:
      - name: Cleanup Stale Branches
        uses: cbrgm/cleanup-stale-branches-action@v1
        with:
          token: ${{ secrets.GITHUB_TOKEN }}
          repository: ${{ github.repository }}
```

### Example Workflow: Run Cleanup on Schedule

This advanced example includes optional configurations:

```yaml
name: Cleanup Stale Branches

on:
  workflow_dispatch:
  schedule:
    - cron: '0 0 * * *' # This schedule runs the workflow at midnight every day

jobs:
  cleanup-stale-branches:
    runs-on: ubuntu-latest
    steps:
      - name: Cleanup Stale Branches
        uses: cbrgm/cleanup-stale-branches-action@v1
        with:
          token: ${{ secrets.GITHUB_TOKEN }}
          repository: ${{ github.repository }}
          ignore-branches: "foobar,release-*"
          ignore-prefixes: "feature/,bugfix/"
          last-commit-age-days: 60
          dry-run: false
          rate-limit: true
```

In this advanced example:

* The action is scheduled to run daily.
* It ignores the branch `foobar` and branches starting with `release-`.
* Branches prefixed with `feature/` or `bugfix/` are not considered for deletion.
* Branches with no commits in the last `60` days are eligible for deletion.
* The action is not in `dry-run` mode, meaning branches will actually be deleted.
* The `rate-limit` check is enabled to prevent exceeding the GitHub API rate limit.

### High-level Functionality

```mermaid
sequenceDiagram
    participant GitHubAction
    participant GitHubAPI

    Note over GitHubAction,GitHubAPI: GitHub Action: cleanup-stale-branches-action

    GitHubAction->>GitHubAPI: Initialize (Token, Repo Info)
    activate GitHubAPI
    GitHubAPI-->>GitHubAction: Repository Validated

    loop For each Branch in Repository
        GitHubAction->>GitHubAPI: Fetch Branch Details
        GitHubAPI-->>GitHubAction: Return Branch Details
        GitHubAction->>GitHubAction: Evaluate Branch Deletion Criteria
        alt Branch Meets Criteria
            alt Dry Run Enabled
                GitHubAction->>GitHubAction: Log Deletable Branch (No Action)
            else Dry Run Disabled
                GitHubAction->>GitHubAPI: Delete Branch
                GitHubAPI-->>GitHubAction: Branch Deleted
            end
        else Branch Does Not Meet Criteria
            GitHubAction->>GitHubAction: Log Skipping Branch
        end
    end

    deactivate GitHubAPI
```

### Local Development

You can build this action from source using `Go`:

```bash
make build
```

## Disclaimer

Usage of this action is entirely at your own risk. While this action has been thoroughly tested, there is always the potential for unexpected behavior, especially in different or complex repository setups. I am not responsible for any damage, data loss, or unexpected code deletion that might occur due to misconfiguration or unexpected behavior of this action. Please ensure you have understood the functionality and have correctly configured the action according to your requirements before using it in a production environment.

## Contributing & License

We welcome and value your contributions to this project! 👍 If you're interested in making improvements or adding features, please refer to our [Contributing Guide](https://github.com/cbrgm/cleanup-stale-branches-action/blob/main/CONTRIBUTING.md). This guide provides comprehensive instructions on how to submit changes, set up your development environment, and more.

Please note that this project is developed in my spare time and is available for free 🕒💻. As an open-source initiative, it is governed by the [Apache 2.0 License](https://github.com/cbrgm/cleanup-stale-branches-action/blob/main/LICENSE). This license outlines your rights and obligations when using, modifying, and distributing this software.

Your involvement, whether it's through code contributions, suggestions, or feedback, is crucial for the ongoing improvement and success of this project. Together, we can ensure it remains a useful and well-maintained resource for everyone 🌍.
