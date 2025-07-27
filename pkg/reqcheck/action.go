package reqcheck

import (
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
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

	return RunWithClient(ctx, cfg, action, pr)
}

// ListChecksFunc is a function type for listing check runs
type ListChecksFunc func(ctx context.Context, sha string, options *github.ListCheckRunsOptions) ([]*github.CheckRun, error)

// RunWithChecksFunc is the same as Run but takes a function for listing checks, useful for testing
func RunWithChecksFunc(ctx context.Context, cfg *Config, action *githubactions.Action, listChecks ListChecksFunc) error {
	action.Infof("Waiting %s before initial check", cfg.InitialDelay)
	time.Sleep(cfg.InitialDelay)

	ghCtx, err := action.Context()
	if err != nil {
		return err
	}

	rules, err := NewRuleset(cfg.RequiredWorkflowPatterns)
	if err != nil {
		return err
	}

	missingRequiredCount := 0
	foundSelf := false
	for {
		checks, err := listChecks(ctx, cfg.TargetSHA, nil)
		if err != nil {
			// Retry if we get an unexpected EOF error, which could be due to proxies.
			if errors.Is(err, io.ErrUnexpectedEOF) {
				action.Infof("Unexpected EOF, retrying...")
				time.Sleep(cfg.PollFrequency)
				continue
			}
			return err
		}
		action.Infof("Got %d checks", len(checks))
		action.Infof("Checks: %q", checkNames(checks))

		requiredSet := lo.SliceToMap(cfg.RequiredWorkflowPatterns, func(item string) (string, bool) { return item, false })
		toCheck := []*github.CheckRun{}
		for _, c := range checks {
			if strings.Contains(c.GetDetailsURL(), fmt.Sprintf("runs/%d/job", ghCtx.RunID)) {
				// skip waiting for this check if this is named the same as another check
				if !foundSelf {
					action.Infof("Skipping check: %q", c.GetName())
					foundSelf = true
				}
				continue
			}
			if found := rules.First(c.GetName()); found != nil {
				toCheck = append(toCheck, c)
				requiredSet[found.String()] = true
			}
		}

		// If required is not found, retry in case the workflow is still being created then fail as there will not be a successful check.
		requiredNotFound := lo.OmitByValues(requiredSet, []bool{true})
		if len(requiredNotFound) > 0 {
			missingRequiredCount++
			if missingRequiredCount > cfg.MissingRequiredRetryCount {
				return fmt.Errorf("required checks not found: %q", lo.Keys(requiredNotFound))
			}
			action.Infof("Required checks not found: %q, continuing another %d times before failing", lo.Keys(requiredNotFound), cfg.MissingRequiredRetryCount-missingRequiredCount)
		}

		// Find failed conclusions, and fail early if there is.
		failed := lo.Filter(toCheck, func(item *github.CheckRun, _ int) bool {
			return slices.Contains(failedConclusions, item.GetConclusion())
		})

		if len(failed) > 0 {
			return fmt.Errorf("required checks failed: %q", checkNames(failed))
		}

		// Wait until all statuses are completed.
		notCompleted := lo.Filter(toCheck, func(item *github.CheckRun, _ int) bool {
			return item.GetStatus() != StatusCompleted
		})

		// Break out of the loop if all checks are completed.
		if len(notCompleted) == 0 {
			action.Infof("All checks completed")
			break
		}

		// sleep and try again.
		action.Infof("Not all checks completed: %q", checkNames(notCompleted))
		action.Infof("Waiting %s before next check", cfg.PollFrequency)
		time.Sleep(cfg.PollFrequency)
	}
	return nil
}

// RunWithClient is the same as Run but takes a pullrequest.Client directly, useful for testing
func RunWithClient(ctx context.Context, cfg *Config, action *githubactions.Action, pr pullrequest.Client) error {
	return RunWithChecksFunc(ctx, cfg, action, pr.ListChecks)
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

type Ruleset []*regexp.Regexp

func NewRuleset(patterns []string) (Ruleset, error) {
	r := make([]*regexp.Regexp, 0, len(patterns))
	for _, p := range patterns {
		re, err := regexp.Compile(p)
		if err != nil {
			return nil, err
		}
		r = append(r, re)
	}
	return r, nil
}

func (r Ruleset) First(test string) *regexp.Regexp {
	for _, re := range r {
		if re.MatchString(test) {
			return re
		}
	}
	return nil
}
