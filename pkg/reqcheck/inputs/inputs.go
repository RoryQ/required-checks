package inputs

const (
	// Token required for the GitHub API
	Token = "TOKEN"

	// RequiredWorkflowPatterns is a yaml list of patterns to check
	RequiredWorkflowPatterns = "REQUIRED_WORKFLOW_PATTERNS"

	// InitialDelaySeconds Initial delay before polling
	InitialDelaySeconds = "INITIAL_DELAY_SECONDS"

	// PollFrequencySeconds Polling frequency
	PollFrequencySeconds = "POLL_FREQUENCY_SECONDS"

	TargetSHA = "TARGET_SHA"

	// Version release version of the action to run
	Version = "version"
)
