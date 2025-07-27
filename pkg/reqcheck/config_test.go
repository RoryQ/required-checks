package reqcheck

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/sethvargo/go-githubactions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/roryq/required-checks/pkg/reqcheck/inputs"
)

func TestConfigFromInputs_DefaultValues(t *testing.T) {
	// Setup
	// Set up minimal environment for defaultTargetSHA
	t.Setenv("GITHUB_SHA", "default-sha")

	action := githubactions.New()

	// Test
	config, err := ConfigFromInputs(action)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, InitialDelayDefault, config.InitialDelay)
	assert.Equal(t, PollFrequencyDefault, config.PollFrequency)
	assert.Equal(t, MissingRequiredRetryCountDefault, config.MissingRequiredRetryCount)
	assert.Empty(t, config.RequiredWorkflowPatterns)
	assert.Equal(t, "default-sha", config.TargetSHA)
}

// In GitHub Actions, inputs are available as environment variables with the format INPUT_<NAME>
func setInput(t *testing.T, input, value string) {
	t.Setenv("INPUT_"+input, value)
}

func unsetInput(t *testing.T, input string) {
	t.Setenv("INPUT_"+input, "")
}

func TestConfigFromInputs(t *testing.T) {
	// Set up minimal environment for defaultTargetSHA
	t.Setenv("GITHUB_SHA", "default-sha")

	tests := map[string]struct {
		Input        string
		Value        string
		SelectConfig func(Config) any
		Expected     any
		AssertError  assert.ErrorAssertionFunc
	}{
		"ValidRequiredWorkflowPattern": {
			Input:        inputs.RequiredWorkflowPatterns,
			Value:        "- pattern1\n- pattern2",
			SelectConfig: func(config Config) any { return config.RequiredWorkflowPatterns },
			Expected:     []string{"pattern1", "pattern2"},
			AssertError:  assert.NoError,
		},
		"ValidConditionalPathWorkflowPatterns": {
			Input: inputs.ConditionalPathWorkflowPatterns,
			Value: `path/to/file*:
  - workflow1
  - workflow2
another/path/**:
  - workflow3`,
			SelectConfig: func(config Config) any { return config.ConditionalPathWorkflowPatterns },
			Expected:     map[string][]string{"path/to/file*": {"workflow1", "workflow2"}, "another/path/**": {"workflow3"}},
			AssertError:  assert.NoError,
		},
		"InvalidGlobConditionalPathWorkflowPatterns": {
			Input:       inputs.ConditionalPathWorkflowPatterns,
			Value:       "[^abc:\n  - workflow1\n  - workflow2",
			AssertError: assert.Error,
		},
		"ValidInitialDelaySeconds": {
			Input:        inputs.InitialDelaySeconds,
			Value:        "30",
			SelectConfig: func(config Config) any { return config.InitialDelay },
			Expected:     30 * time.Second,
			AssertError:  assert.NoError,
		},
		"InvalidInitialDelaySeconds": {
			Input:        inputs.InitialDelaySeconds,
			Value:        "not-a-number",
			SelectConfig: func(config Config) any { return config.InitialDelay },
			Expected:     InitialDelayDefault,
			AssertError:  assert.NoError, // Invalid numbers should not cause errors, just warnings
		},
		"ValidPollFrequencySeconds": {
			Input:        inputs.PollFrequencySeconds,
			Value:        "60",
			SelectConfig: func(config Config) any { return config.PollFrequency },
			Expected:     60 * time.Second,
			AssertError:  assert.NoError,
		},
		"InvalidPollFrequencySeconds": {
			Input:        inputs.PollFrequencySeconds,
			Value:        "not-a-number",
			SelectConfig: func(config Config) any { return config.PollFrequency },
			Expected:     PollFrequencyDefault,
			AssertError:  assert.NoError, // Invalid numbers should not cause errors, just warnings
		},
		"ValidMissingRequiredRetryCount": {
			Input:        inputs.MissingRequiredRetryCount,
			Value:        "5",
			SelectConfig: func(config Config) any { return config.MissingRequiredRetryCount },
			Expected:     5,
			AssertError:  assert.NoError,
		},
		"InvalidMissingRequiredRetryCount": {
			Input:        inputs.MissingRequiredRetryCount,
			Value:        "not-a-number",
			SelectConfig: func(config Config) any { return config.MissingRequiredRetryCount },
			Expected:     MissingRequiredRetryCountDefault,
			AssertError:  assert.NoError, // Invalid numbers should not cause errors, just warnings
		},
		"ValidTargetSHA": {
			Input:        inputs.TargetSHA,
			Value:        "custom-sha",
			SelectConfig: func(config Config) any { return config.TargetSHA },
			Expected:     "custom-sha",
			AssertError:  assert.NoError,
		},
	}

	action := githubactions.New()
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Set the test input
			setInput(t, tt.Input, tt.Value)

			config, err := ConfigFromInputs(action)
			tt.AssertError(t, err)
			if err != nil {
				return
			}
			require.NotNil(t, config)
			assert.Equal(t, tt.Expected, tt.SelectConfig(*config))
		})
	}
}

func TestConfigFromInputs_InvalidYAML(t *testing.T) {
	// Setup
	setInput(t, inputs.RequiredWorkflowPatterns, "invalid: yaml: [")

	action := githubactions.New()

	// Test
	config, err := ConfigFromInputs(action)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "yaml")
	assert.Nil(t, config)
}

func TestDefaultTargetSHA_FromInput(t *testing.T) {
	// Setup
	setInput(t, inputs.TargetSHA, "input-sha")

	action := githubactions.New()

	// Test
	sha, err := defaultTargetSHA(action)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "input-sha", sha)
}

func TestDefaultTargetSHA_FromPullRequest(t *testing.T) {
	// Setup
	// Create a temporary file with PR event data
	eventFile, err := os.CreateTemp("", "pr-event-*.json")
	require.NoError(t, err)
	defer os.Remove(eventFile.Name())

	prEvent := map[string]interface{}{
		"pull_request": map[string]interface{}{
			"head": map[string]interface{}{
				"sha": "pr-head-sha",
			},
		},
	}

	eventData, err := json.Marshal(prEvent)
	require.NoError(t, err)

	_, err = eventFile.Write(eventData)
	require.NoError(t, err)
	eventFile.Close()

	// Set up environment
	t.Setenv("GITHUB_EVENT_PATH", eventFile.Name())
	t.Setenv("GITHUB_SHA", "commit-sha")

	action := githubactions.New()

	// Test
	sha, err := defaultTargetSHA(action)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "pr-head-sha", sha)
}

func TestDefaultTargetSHA_FromCommit(t *testing.T) {
	// Setup
	// Create a temporary file with non-PR event data
	eventFile, err := os.CreateTemp("", "commit-event-*.json")
	require.NoError(t, err)
	defer os.Remove(eventFile.Name())

	commitEvent := map[string]interface{}{}

	eventData, err := json.Marshal(commitEvent)
	require.NoError(t, err)

	_, err = eventFile.Write(eventData)
	require.NoError(t, err)
	eventFile.Close()

	// Set up environment
	t.Setenv("GITHUB_EVENT_PATH", eventFile.Name())
	t.Setenv("GITHUB_SHA", "commit-sha")

	action := githubactions.New()

	// Test
	sha, err := defaultTargetSHA(action)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "commit-sha", sha)
}
