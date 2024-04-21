package reqcheck

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/go-github/v61/github"
	"github.com/samber/lo"
	"github.com/sethvargo/go-githubactions"
	"slices"

	"github.com/roryq/required-checks/pkg/pullrequest"
)

func Run(ctx context.Context, cfg *Config, action *githubactions.Action, gh *github.Client) error {
	pr, err := pullrequest.NewClient(action, gh)
	if err != nil {
		return err
	}

	action.Infof("Waiting %s before initial check", cfg.InitialDelay)
	time.Sleep(cfg.InitialDelay)

	ghCtx, err := action.Context()
	if err != nil {
		return err
	}

	for {
		checks, err := pr.ListChecks(ctx, cfg.TargetSHA, nil)
		if err != nil {
			return err
		}
		action.Infof("Got %d checks", len(checks))
		action.Infof("Checks: %q", checkNames(checks))

		requiredSet := lo.SliceToMap(cfg.RequiredWorkflowPatterns, func(item string) (string, bool) { return item, false })
		toCheck := []*github.CheckRun{}
		for _, c := range checks {
			if strings.Contains(c.GetDetailsURL(), fmt.Sprintf("runs/%d/job", ghCtx.RunID)) {
				// skip waiting for this check if this is named the same as another check
				action.Infof("Skipping check: %q", c.GetName())
				continue
			}
			if _, ok := requiredSet[c.GetName()]; ok {
				toCheck = append(toCheck, c)
				requiredSet[c.GetName()] = true
			}
		}

		requiredNotFound := lo.OmitByValues(requiredSet, []bool{true})
		if len(requiredNotFound) > 0 {
			return fmt.Errorf("required checks not found: %q in %q", lo.Keys(requiredNotFound))
		}

		// any failed
		failed := lo.Filter(toCheck, func(item *github.CheckRun, _ int) bool {
			return slices.Contains(failedConclusions, item.GetConclusion())
		})

		if len(failed) > 0 {
			return fmt.Errorf("required checks failed: %q", checkNames(failed))
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

var failedConclusions = []string{ConclusionFailure, ConclusionCancelled, ConclusionTimedOut}
