package pullrequest

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/google/go-github/v61/github"
	"github.com/sethvargo/go-githubactions"
)

type Client struct {
	Owner  string
	Repo   string
	Number Optional[int]
	ctx    *githubactions.GitHubContext
	gh     *github.Client
}

type Optional[T any] = sql.Null[T]

func (pr Client) ListChecks(ctx context.Context, sha string, options *github.ListCheckRunsOptions) ([]*github.CheckRun, error) {
	var checks []*github.CheckRun
	for {
		checksPage, resp, err := pr.gh.Checks.ListCheckRunsForRef(ctx, pr.Owner, pr.Repo, sha, options)
		if err != nil {
			return nil, err
		}
		checks = append(checks, checksPage.CheckRuns...)
		if resp.NextPage == 0 {
			break
		}
		if options == nil {
			options = &github.ListCheckRunsOptions{
				ListOptions: github.ListOptions{
					Page: resp.NextPage,
				},
			}
		}
		options.Page = resp.NextPage
	}
	return checks, nil
}

func (pr Client) ListFiles(ctx context.Context, options *github.ListOptions) ([]*github.CommitFile, error) {
	// not supported
	if !pr.Number.Valid {
		return nil, nil
	}
	var files []*github.CommitFile
	for {
		filesPage, resp, err := pr.gh.PullRequests.ListFiles(ctx, pr.Owner, pr.Repo, pr.Number.V, options)
		if err != nil {
			return nil, err
		}
		files = append(files, filesPage...)
		if resp.NextPage == 0 {
			break
		}
		if options == nil {
			options = &github.ListOptions{
				Page: resp.NextPage,
			}
		}
		options.Page = resp.NextPage
	}
	return files, nil
}

func NewClient(action *githubactions.Action, gh *github.Client) (Client, error) {
	ctx, err := action.Context()
	if err != nil {
		return Client{}, err
	}

	var number Optional[int]
	switch ctx.EventName {
	case "merge_group":
		action.Debugf("Skipping path globs for merge_group")
	default:
		n, err := getPRNumber(ctx.Event)
		if err != nil {
			return Client{}, err
		}
		number = Optional[int]{V: n, Valid: true}
	}

	owner, repo := getRepo(action, ctx.Event)
	action.Debugf("action context: %s %s", owner, repo)
	return Client{
		Owner:  owner,
		Repo:   repo,
		Number: number,
		ctx:    ctx,
		gh:     gh,
	}, nil
}

func getRepo(action *githubactions.Action, event map[string]any) (string, string) {
	splitRepo := func(name string) (string, string) {
		split := strings.Split(name, "/")
		return split[0], split[1]
	}

	if fullName := action.Getenv("GITHUB_REPOSITORY"); fullName != "" {
		splitRepo(fullName)
	}

	if fullName, ok := event["repository"].(map[string]any)["full_name"]; ok {
		return splitRepo(fullName.(string))
	}
	return "", ""
}

func getPRNumber(event map[string]any) (int, error) {
	getNumber := func(eventName string) (int, error) {
		eventField, ok := event[eventName]
		if !ok {
			return 0, errors.New("incorrect event type")
		}

		number, ok := eventField.(map[string]any)["number"]
		if !ok {
			return 0, errors.New("cannot get pull_request number")
		}
		return int(number.(float64)), nil
	}

	num, err := getNumber("pull_request")
	if err != nil {
		return getNumber("issue")
	}
	return num, err
}
