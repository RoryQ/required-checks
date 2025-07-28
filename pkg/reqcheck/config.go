package reqcheck

import (
	"strconv"
	"time"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/niemeyer/pretty"
	"github.com/sethvargo/go-githubactions"
	"gopkg.in/yaml.v3"

	"github.com/roryq/required-checks/pkg/reqcheck/inputs"
)

type Config struct {
	RequiredWorkflowPatterns        []string
	ConditionalPathWorkflowPatterns map[string][]string
	InitialDelay                    time.Duration
	PollFrequency                   time.Duration
	MissingRequiredRetryCount       int
	TargetSHA                       string
}

const (
	InitialDelayDefault              = 15 * time.Second
	PollFrequencyDefault             = 30 * time.Second
	MissingRequiredRetryCountDefault = 2
)

func ConfigFromInputs(action *githubactions.Action) (*Config, error) {
	action.Infof("Reading Config From Inputs")
	c := Config{
		InitialDelay:                    InitialDelayDefault,
		PollFrequency:                   PollFrequencyDefault,
		ConditionalPathWorkflowPatterns: map[string][]string{},
		MissingRequiredRetryCount:       MissingRequiredRetryCountDefault,
	}
	requiredWorkflowPatterns := action.GetInput(inputs.RequiredWorkflowPatterns)
	if requiredWorkflowPatterns != "" {
		if err := yaml.Unmarshal([]byte(requiredWorkflowPatterns), &c.RequiredWorkflowPatterns); err != nil {
			return nil, err
		}
	}

	pathPatterns := action.GetInput(inputs.ConditionalPathWorkflowPatterns)
	if pathPatterns != "" {
		if err := yaml.Unmarshal([]byte(pathPatterns), &c.ConditionalPathWorkflowPatterns); err != nil {
			return nil, err
		}
		for key, _ := range c.ConditionalPathWorkflowPatterns {
			if !doublestar.ValidatePathPattern(key) {
				action.Warningf("Invalid workflow pattern: %s", key)
			}
		}
	}

	if initialDelaySeconds := action.GetInput(inputs.InitialDelaySeconds); initialDelaySeconds != "" {
		if ids, err := strconv.Atoi(initialDelaySeconds); err != nil {
			action.Warningf("Failed to parse InitialDelaySeconds: %s", err)
		} else {
			c.InitialDelay = time.Duration(ids) * time.Second
		}
	}

	if pollFrequencySeconds := action.GetInput(inputs.PollFrequencySeconds); pollFrequencySeconds != "" {
		if pfs, err := strconv.Atoi(pollFrequencySeconds); err != nil {
			action.Warningf("Failed to parse PollFrequencySeconds: %s", err)
		} else {
			c.PollFrequency = time.Duration(pfs) * time.Second
		}
	}

	if missingRequiredRetryCount := action.GetInput(inputs.MissingRequiredRetryCount); missingRequiredRetryCount != "" {
		if mrrc, err := strconv.Atoi(missingRequiredRetryCount); err != nil {
			action.Warningf("Failed to parse MissingRequiredRetryCount: %s", err)
		} else {
			c.MissingRequiredRetryCount = mrrc
		}
	}

	var err error
	c.TargetSHA, err = defaultTargetSHA(action)
	if err != nil {
		return nil, err
	}

	action.Infof("Config: %s", pretty.Sprint(c))

	return &c, nil
}

// equivalent of ${{ github.event.pull_request.head.sha || github.sha }}
func defaultTargetSHA(action *githubactions.Action) (string, error) {
	targetSha := action.GetInput(inputs.TargetSHA)
	if targetSha != "" {
		action.Infof("Target SHA: %s", targetSha)
		return targetSha, nil
	}
	ctx, err := action.Context()
	if err != nil {
		return "", err
	}

	sha := ctx.SHA

	if pr, ok := ctx.Event["pull_request"]; ok {
		sha = pr.(map[string]any)["head"].(map[string]any)["sha"].(string)
		action.Infof("Pull Request SHA: %s", sha)
	} else {
		action.Infof("Commit SHA: %s", sha)
	}

	return sha, nil
}
