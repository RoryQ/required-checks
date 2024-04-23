package pullrequest

import (
	"context"
	"strings"

	"github.com/google/go-github/v61/github"
	"github.com/sethvargo/go-githubactions"
)

type Client struct {
	Owner string
	Repo  string
	ctx   *githubactions.GitHubContext
	gh    *github.Client
}

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

func NewClient(action *githubactions.Action, gh *github.Client) (Client, error) {
	ctx, err := action.Context()
	if err != nil {
		return Client{}, err
	}

	owner, repo := getRepo(action, ctx.Event)
	action.Debugf("action context: %s %s", owner, repo)
	return Client{
		Owner: owner,
		Repo:  repo,
		ctx:   ctx,
		gh:    gh,
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
