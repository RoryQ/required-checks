package reqcheck

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/google/go-github/v61/github"
	"github.com/samber/lo"
	"github.com/sethvargo/go-githubactions"

	"github.com/roryq/required-checks/pkg/pullrequest"
)

func Run(ctx context.Context, cfg *Config, action *githubactions.Action, gh *github.Client) error {
	pr, err := pullrequest.NewClient(action, gh)
	if err != nil {
		return err
	}

	action.Infof("Waiting %s before initial check", cfg.InitialDelay)
	time.Sleep(cfg.InitialDelay)

	for {
		checks, err := pr.ListChecks(ctx, cfg.TargetSHA, nil)
		if err != nil {
			return err
		}
		action.Infof("Got %d checks", len(checks))
		action.Debugf("Checks: %q", checkNames(checks))

		requiredSet := lo.SliceToMap(cfg.RequiredWorkflowPatterns, func(item string) (string, struct{}) { return item, struct{}{} })
		toCheck := []*github.CheckRun{}
		for _, c := range checks {
			if _, ok := requiredSet[c.GetName()]; ok {
				toCheck = append(toCheck, c)
				delete(requiredSet, c.GetName())
			}
		}

		if len(requiredSet) > 0 {
			return fmt.Errorf("requiredSet checks not found: %q", keys(requiredSet))
		}

		// any failed
		failed := lo.Filter(toCheck, func(item *github.CheckRun, _ int) bool {
			return slices.Contains(failed, item.GetConclusion())
		})

		if len(failed) > 0 {
			return fmt.Errorf("requiredSet checks failed: %q", checkNames(failed))
		}

		// not completed
		notCompleted := lo.Filter(toCheck, func(item *github.CheckRun, _ int) bool {
			return item.GetStatus() != StatusCompleted
		})

		if len(notCompleted) == 0 {
			action.Infof("All checks completed")
			break
		}

		action.Infof("Not all checks completed: %q", checkNames(notCompleted))
		action.Infof("Waiting %s before next check", cfg.PollFrequency)
		time.Sleep(cfg.PollFrequency)
	}
	return nil
}

func keys(m map[string]struct{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func checkNames(checks []*github.CheckRun) []string {
	names := make([]string, 0, len(checks))
	for _, c := range checks {
		names = append(names, c.GetName())
	}
	return names
}

// Conclusion: action_required, cancelled, failure, neutral, success, skipped, stale, timed_out
const (
	ConclusionActionRequired = "action_required"
	ConclusionCancelled      = "cancelled"
	ConclusionFailure        = "failure"
	ConclusionNeutral        = "neutral"
	ConclusionSuccess        = "success"
	ConclusionSkipped        = "skipped"
	ConclusionStale          = "stale"
	ConclusionTimedOut       = "timed_out"
)

// StatusL queued, in_progress, completed, waiting, requested, pending
const (
	StatusQueued     = "queued"
	StatusInProgress = "in_progress"
	StatusCompleted  = "completed"
	StatusWaiting    = "waiting"
	StatusRequested  = "requested"
	StatusPending    = "pending"
)

var failed = []string{ConclusionFailure, ConclusionCancelled, ConclusionTimedOut}
