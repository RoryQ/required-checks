# Required Checks

## Define required checks inside your repository

## Features

- [x] Fail if any configured checks fail
- [x] Fail if a configured check fails to report
- [x] Define check name patterns using regular expressions
- [x] Require checks when certain files are changed

## Configuration

```yaml
on:
  pull_request:

name: Required Checks
jobs:
  required-checks:
    runs-on: ubuntu-latest
    steps:
    - name: Wait for required checks
      uses: roryq/required-checks@master
      with:
        # required_workflow_patterns is a yaml list of regex patterns to check
        required_workflow_patterns: |
          # will match any check with tests in its name
          - tests
          # will match either markdown-lint or yaml-lint
          - (markdown-lint|yaml-lint)

        # A yaml dictionary of path globs and regex patterns. If a commit file matches a path glob then the corresponding
        # regex patterns will be added to the list of workflows to check.
        conditional_path_workflow_patterns: |
          "**/*.go": 
            - "go-unit-tests"
          "**/*.sql":
            - "validate-migrations"
          
        # GitHub token
        token: ${{ secrets.GITHUB_TOKEN }}
        # number of seconds to wait before starting the first poll
        initial_delay_seconds: 15
        # number of seconds to wait between polls
        poll_frequency_seconds: 30
        # number of times to retry if a required check is missing. 
        # This is useful in cases where the workflow is still being created.
        missing_required_retry_count: 3
        # target sha that the checks have been run against. Defaults to ${{ github.event.pull_request.head.sha || github.sha }}
        target_sha: ${{ github.event.pull_request.head.sha || github.sha }}

```
