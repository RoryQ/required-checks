package inputs

const (
	// Token required for the GitHub API
	Token = "TOKEN"

	// RequiredWorkflowPatterns is a yaml list of patterns to check
	RequiredWorkflowPatterns = "REQUIRED_WORKFLOW_PATTERNS"

	// ConditionalPathWorkflowPatterns path globs and patterns defining optional workflows to check for certain file changes.
	ConditionalPathWorkflowPatterns = "CONDITIONAL_PATH_WORKFLOW_PATTERNS"

	// InitialDelaySeconds Initial delay before polling
	InitialDelaySeconds = "INITIAL_DELAY_SECONDS"

	// PollFrequencySeconds Polling frequency
	PollFrequencySeconds = "POLL_FREQUENCY_SECONDS"

	TargetSHA = "TARGET_SHA"

	// MissingRequiredRetryCount is the number of times to retry if a required check is missing, for cases where the workflow is still being created.
	MissingRequiredRetryCount = "MISSING_REQUIRED_RETRY_COUNT"

	// Version release version of the action to run
	Version = "VERSION"
)
