package reqcheck

import (
	"context"

	"github.com/google/go-github/v61/github"
	"github.com/sanity-io/litter"
	"github.com/sethvargo/go-githubactions"

	"github.com/roryq/required-checks/pkg/pullrequest"
)

func Run(ctx context.Context, cfg *Config, action *githubactions.Action, gh *github.Client) error {
	pr, err := pullrequest.NewClient(action, gh)
	if err != nil {
		return err
	}

	checks, err := pr.ListChecks(ctx, nil)
	litter.Dump(checks)

	return nil
}
