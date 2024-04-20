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
	InitialDelay time.Duration
	PollFrequency time.Duration
}

const (
	InitialDelayDefault = 10 * time.Second
	PollFrequencyDefault = 15 * time.Second
)

func ConfigFromInputs(action *githubactions.Action) (*Config, error) {
	action.Infof("Reading Config From Inputs")
	c := Config{
		InitialDelay:  InitialDelayDefault,
		PollFrequency: PollFrequencyDefault,
	}
	requiredWorkflowPatterns := action.GetInput(inputs.RequiredWorkflowPatterns)
	if requiredWorkflowPatterns == "" {
		return &c, nil
	}

	if err := yaml.Unmarshal([]byte(requiredWorkflowPatterns), &c.RequiredWorkflowPatterns); err != nil {
		return nil, err
	}

	action.Infof("Config: %s", pretty.Sprint(c))

	return &c, nil
}
