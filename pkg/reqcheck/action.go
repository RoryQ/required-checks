package reqcheck

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/bmatcuk/doublestar/v4"
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

	return run(ctx, cfg, action, pr)
}

// run is the same as Run but takes a function for listing checks, useful for testing
func run(ctx context.Context, cfg *Config, action *githubactions.Action, pr PRClient) error {
	action.Infof("Waiting %s before initial check", cfg.InitialDelay)
	time.Sleep(cfg.InitialDelay)

	ghCtx, err := action.Context()
	if err != nil {
		return err
	}

	workflowPatterns := cfg.RequiredWorkflowPatterns
	pathWorkflowPatterns, err := getConditionalPathPatterns(ctx, cfg, action, pr)
	if err != nil {
		return err
	}

	workflowPatterns = lo.Uniq(append(workflowPatterns, pathWorkflowPatterns...))
	rules, err := NewRuleset(workflowPatterns)
	if err != nil {
		return err
	}

	missingRequiredCount := 0
	foundSelf := false
	for {
		checks, err := pr.ListChecks(ctx, cfg.TargetSHA, nil)
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

		requiredSet := lo.SliceToMap(workflowPatterns, func(item string) (string, bool) { return item, false })
		toCheck := []*github.CheckRun{}
		for _, c := range checks {
			if strings.Contains(c.GetDetailsURL(), fmt.Sprintf("runs/%d/job", ghCtx.RunID)) {
				// skip waiting for this check if this is named the same as another check
				if !foundSelf && c.GetName() == ghCtx.Job {
					action.Infof("Skipping check: %q", c.GetName())
					foundSelf = true
					continue
				}
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
				return fmt.Errorf("required checks not found: %q", sortStrings(lo.Keys(requiredNotFound)))
			}
			action.Infof("Required checks not found: %q, continuing another %d times before failing", lo.Keys(requiredNotFound), cfg.MissingRequiredRetryCount-missingRequiredCount)
			action.Infof("Waiting %s before next check", cfg.PollFrequency)
			time.Sleep(cfg.PollFrequency)
			continue
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

func getConditionalPathPatterns(ctx context.Context, cfg *Config, action *githubactions.Action, pr PRClient) ([]string, error) {
	if len(cfg.ConditionalPathWorkflowPatterns) == 0 {
		return nil, nil
	}

	fileNames, err := listPullRequestFiles(ctx, pr)
	if err != nil {
		return nil, err
	}
	matched := lo.Filter(lo.Keys(cfg.ConditionalPathWorkflowPatterns), func(pathGlob string, _ int) bool {
		for _, name := range fileNames {
			if matched, _ := doublestar.Match(pathGlob, name); matched {
				action.Infof("Matched path glob [%s] with file: %s", pathGlob, name)
				action.Infof("Adding checks to required: %q", cfg.ConditionalPathWorkflowPatterns[pathGlob])
				return true
			}
		}
		return false
	})
	return lo.Flatten(lo.Values(lo.PickByKeys(cfg.ConditionalPathWorkflowPatterns, matched))), nil
}

type PRClient interface {
	ListChecks(ctx context.Context, sha string, options *github.ListCheckRunsOptions) ([]*github.CheckRun, error)
	ListFiles(ctx context.Context, options *github.ListOptions) ([]*github.CommitFile, error)
}

func checkNames(checks []*github.CheckRun) []string {
	names := make([]string, 0, len(checks))
	for _, c := range checks {
		names = append(names, c.GetName())
	}
	return names
}

func listPullRequestFiles(ctx context.Context, pr PRClient) ([]string, error) {
	files, err := pr.ListFiles(ctx, nil)
	if isNotFoundError(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	fileNames := lo.Map(files, func(item *github.CommitFile, _ int) string { return item.GetFilename() })
	return fileNames, nil
}

func isNotFoundError(err error) bool {
	ghe := new(github.ErrorResponse)
	if errors.As(err, &ghe) {
		if ghe.Response.StatusCode == http.StatusNotFound {
			return true
		}
	}
	return false
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

func sortStrings(slice []string) []string {
	sort.Strings(slice)
	return slice
}
