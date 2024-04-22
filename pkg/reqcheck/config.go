package reqcheck

import (
	"time"

	"github.com/niemeyer/pretty"
	"github.com/sethvargo/go-githubactions"
	"gopkg.in/yaml.v3"

	"github.com/roryq/required-checks/pkg/reqcheck/inputs"
)

type Config struct {
	RequiredWorkflowPatterns []string
	InitialDelay             time.Duration
	PollFrequency            time.Duration
	TargetSHA                string
}

const (
	InitialDelayDefault  = 15 * time.Second
	PollFrequencyDefault = 30 * time.Second
)

func ConfigFromInputs(action *githubactions.Action) (*Config, error) {
	action.Infof("Reading Config From Inputs")
	c := Config{
		InitialDelay:  InitialDelayDefault,
		PollFrequency: PollFrequencyDefault,
	}
	requiredWorkflowPatterns := action.GetInput(inputs.RequiredWorkflowPatterns)
	if requiredWorkflowPatterns != "" {
		if err := yaml.Unmarshal([]byte(requiredWorkflowPatterns), &c.RequiredWorkflowPatterns); err != nil {
			return nil, err
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
