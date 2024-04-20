package inputs

const (
	// Token required for the GitHub API
	Token = "token"

	// RequiredWorkflowPatterns is a yaml list of patterns to check
	RequiredWorkflowPatterns = "required-workflow-patterns"

	// InitialDelaySeconds Initial delay before polling
	InitialDelaySeconds = "initial-delay-seconds"

	// PollFrequencySeconds Polling frequency
	PollFrequencySeconds = "poll-frequency-seconds"

	// Version release version of the action to run
	Version = "version"
)
