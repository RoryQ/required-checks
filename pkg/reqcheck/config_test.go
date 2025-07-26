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

func TestConfigFromInputs_ValidInputs(t *testing.T) {
	// Setup
	setInput(t, inputs.RequiredWorkflowPatterns, `- pattern1
- pattern2`)
	setInput(t, inputs.InitialDelaySeconds, "30")
	setInput(t, inputs.PollFrequencySeconds, "60")
	setInput(t, inputs.MissingRequiredRetryCount, "5")
	setInput(t, inputs.TargetSHA, "custom-sha")

	action := githubactions.New()

	// Test
	config, err := ConfigFromInputs(action)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, []string{"pattern1", "pattern2"}, config.RequiredWorkflowPatterns)
	assert.Equal(t, 30*time.Second, config.InitialDelay)
	assert.Equal(t, 60*time.Second, config.PollFrequency)
	assert.Equal(t, 5, config.MissingRequiredRetryCount)
	assert.Equal(t, "custom-sha", config.TargetSHA)
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

func TestConfigFromInputs_InvalidNumbers(t *testing.T) {
	testCases := []struct {
		name      string
		inputName string
		inputVal  string
	}{
		{
			name:      "Invalid InitialDelaySeconds",
			inputName: inputs.InitialDelaySeconds,
			inputVal:  "not-a-number",
		},
		{
			name:      "Invalid PollFrequencySeconds",
			inputName: inputs.PollFrequencySeconds,
			inputVal:  "not-a-number",
		},
		{
			name:      "Invalid MissingRequiredRetryCount",
			inputName: inputs.MissingRequiredRetryCount,
			inputVal:  "not-a-number",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup
			// Clear previous environment variables
			unsetInput(t, inputs.InitialDelaySeconds)
			unsetInput(t, inputs.PollFrequencySeconds)
			unsetInput(t, inputs.MissingRequiredRetryCount)

			// Set the test input
			setInput(t, tc.inputName, tc.inputVal)

			// Set up minimal environment for defaultTargetSHA
			t.Setenv("GITHUB_SHA", "default-sha")

			action := githubactions.New()

			// Test
			config, err := ConfigFromInputs(action)

			// Assert
			require.NoError(t, err) // Invalid numbers should not cause errors, just warnings
			assert.NotNil(t, config)

			// Check that default values are used for invalid inputs
			switch tc.inputName {
			case inputs.InitialDelaySeconds:
				assert.Equal(t, InitialDelayDefault, config.InitialDelay)
			case inputs.PollFrequencySeconds:
				assert.Equal(t, PollFrequencyDefault, config.PollFrequency)
			case inputs.MissingRequiredRetryCount:
				assert.Equal(t, MissingRequiredRetryCountDefault, config.MissingRequiredRetryCount)
			}
		})
	}
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
