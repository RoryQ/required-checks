package reqcheck

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/google/go-github/v61/github"
	"github.com/sethvargo/go-githubactions"
	"github.com/stretchr/testify/assert"
	"slices"

	"github.com/roryq/required-checks/pkg/xassert"
)

func TestRun(t *testing.T) {
	testCases := map[string]struct {
		config              *Config
		checkRuns           []*github.CheckRun
		prFiles             []*github.CommitFile
		listChecksError     error
		assertError         assert.ErrorAssertionFunc
		expectedOutputLines []string
		progressiveChecks   bool
	}{
		"all required checks pass": {
			config: &Config{
				RequiredWorkflowPatterns:  []string{"required-check-1", "required-check-2"},
				MissingRequiredRetryCount: 1,
				TargetSHA:                 "test-sha",
			},
			checkRuns: []*github.CheckRun{
				{
					Name:       github.String("required-check-1"),
					Status:     github.String(StatusCompleted),
					Conclusion: github.String(ConclusionSuccess),
				},
				{
					Name:       github.String("required-check-2"),
					Status:     github.String(StatusCompleted),
					Conclusion: github.String(ConclusionSuccess),
				},
			},
			assertError: assert.NoError,
			expectedOutputLines: []string{
				"Waiting 1ms before initial check",
				"Got 2 checks",
				`Checks: ["required-check-1" "required-check-2"]`,
				"All checks completed",
			},
			progressiveChecks: false,
		},
		"required check fails": {
			config: &Config{
				RequiredWorkflowPatterns:  []string{"required-check-1", "required-check-2"},
				MissingRequiredRetryCount: 1,
				TargetSHA:                 "test-sha",
			},
			checkRuns: []*github.CheckRun{
				{
					Name:       github.String("required-check-1"),
					Status:     github.String(StatusCompleted),
					Conclusion: github.String(ConclusionSuccess),
				},
				{
					Name:       github.String("required-check-2"),
					Status:     github.String(StatusCompleted),
					Conclusion: github.String(ConclusionFailure),
				},
			},
			assertError:       xassert.ErrorContains(`required checks failed: ["required-check-2"]`),
			progressiveChecks: false,
		},
		"required check missing": {
			config: &Config{
				RequiredWorkflowPatterns:  []string{"required-check-1", "missing-check"},
				MissingRequiredRetryCount: 0,
				TargetSHA:                 "test-sha",
			},
			checkRuns: []*github.CheckRun{
				{
					Name:       github.String("required-check-1"),
					Status:     github.String(StatusCompleted),
					Conclusion: github.String(ConclusionSuccess),
				},
			},
			assertError: xassert.ErrorContains(`required checks not found: ["missing-check"]`),
			expectedOutputLines: []string{
				"Waiting 1ms before initial check",
				"Got 1 checks",
				`Checks: ["required-check-1"]`,
			},
			progressiveChecks: false,
		},
		"unexpected EOF error": {
			config: &Config{
				RequiredWorkflowPatterns:  []string{"required-check-1"},
				MissingRequiredRetryCount: 1,
				TargetSHA:                 "test-sha",
			},
			listChecksError: io.ErrUnexpectedEOF,
			checkRuns: []*github.CheckRun{
				{
					Name:       github.String("required-check-1"),
					Status:     github.String(StatusCompleted),
					Conclusion: github.String(ConclusionSuccess),
				},
			},
			assertError: assert.NoError,
			expectedOutputLines: []string{
				"Waiting 1ms before initial check",
				"Unexpected EOF, retrying...",
				"Got 1 checks",
				`Checks: ["required-check-1"]`,
				"All checks completed",
			},
			progressiveChecks: false,
		},
		"check not completed": {
			config: &Config{
				RequiredWorkflowPatterns:  []string{"required-check-1"},
				MissingRequiredRetryCount: 1,
				TargetSHA:                 "test-sha",
			},
			checkRuns: []*github.CheckRun{
				{
					Name:   github.String("required-check-1"),
					Status: github.String(StatusInProgress),
				},
				{
					Name:       github.String("required-check-1"),
					Status:     github.String(StatusCompleted),
					Conclusion: github.String(ConclusionSuccess),
				},
			},
			assertError: assert.NoError,
			expectedOutputLines: []string{
				`Waiting 1ms before initial check`,
				`Got 1 checks`,
				`Checks: ["required-check-1"]`,
				`Not all checks completed: ["required-check-1"]`,
				`Waiting 1ms before next check`,
				`Got 1 checks`,
				`Checks: ["required-check-1"]`,
				`All checks completed`,
			},
			progressiveChecks: true,
		},
		"path based checklist is activated": {
			config: &Config{
				ConditionalPathWorkflowPatterns: map[string][]string{"main.go": {"go unit tests"}},
			},
			checkRuns: []*github.CheckRun{
				{
					Name:   github.String("go unit tests"),
					Status: github.String(StatusInProgress),
				},
				{
					Name:       github.String("go unit tests"),
					Status:     github.String(StatusCompleted),
					Conclusion: github.String(ConclusionSuccess),
				},
			},
			prFiles: []*github.CommitFile{
				{Filename: github.String("main.go")},
			},
			assertError:       assert.NoError,
			progressiveChecks: true,
			expectedOutputLines: []string{
				"Matched path glob [main.go] with file: main.go",
				`Got 1 checks`,
				`Checks: ["go unit tests"]`,
				`Not all checks completed: ["go unit tests"]`,
				`Waiting 1ms before next check`,
				`Got 1 checks`,
				`Checks: ["go unit tests"]`,
				`All checks completed`,
			},
		},
		"path based checklist fails if missing": {
			config: &Config{
				ConditionalPathWorkflowPatterns: map[string][]string{"main.go": {"go unit tests"}},
			},
			checkRuns: []*github.CheckRun{},
			prFiles: []*github.CommitFile{
				{Filename: github.String("main.go")},
			},
			assertError: xassert.ErrorContains(`required checks not found: ["go unit tests"]`),
			expectedOutputLines: []string{
				`Matched path glob [main.go] with file: main.go`,
				`Got 0 checks`,
				`Checks: []`,
			},
		},
		"path based checklist combines with required checks": {
			config: &Config{
				ConditionalPathWorkflowPatterns: map[string][]string{"main.go": {"go unit tests"}},
				RequiredWorkflowPatterns:        []string{"required-check"},
			},
			checkRuns: []*github.CheckRun{
				{
					Name:       github.String("required-check"),
					Status:     github.String(StatusCompleted),
					Conclusion: github.String(ConclusionSuccess),
				},
				{
					Name:       github.String("go unit tests"),
					Status:     github.String(StatusCompleted),
					Conclusion: github.String(ConclusionSuccess),
				},
			},
			prFiles: []*github.CommitFile{
				{Filename: github.String("main.go")},
			},
			assertError: assert.NoError,
			expectedOutputLines: []string{
				"Matched path glob [main.go] with file: main.go",
				`Got 2 checks`,
				`Checks: ["required-check" "go unit tests"]`,
				`All checks completed`,
			},
		},
	}

	for name, tc := range testCases {
		// set default freq
		tc.config.InitialDelay = time.Millisecond
		tc.config.PollFrequency = time.Millisecond

		t.Run(name, func(t *testing.T) {
			// Setup
			action, output := setupAction("pull-request.opened")

			// Create a mock pullrequest client
			mockPRClient := setupMockPRClient(tc.checkRuns, tc.listChecksError, tc.progressiveChecks, tc.prFiles)

			// Run the function
			err := run(context.Background(), tc.config, action, mockPRClient)

			// Check error
			tc.assertError(t, err)

			// Check output
			// t.Log(output.String())
			outputStr := output.String()
			for _, expectedLine := range tc.expectedOutputLines {
				assert.Contains(t, outputStr, expectedLine)
			}
		})
	}
}

// mockPullRequestClient is a mock implementation of the pullrequest.Client
type mockPullRequestClient struct {
	ListChecksFunc func(ctx context.Context, sha string, options *github.ListCheckRunsOptions) ([]*github.CheckRun, error)
	ListFilesFunc  func(ctx context.Context, options *github.ListOptions) ([]*github.CommitFile, error)
}

func (m *mockPullRequestClient) ListChecks(ctx context.Context, sha string, options *github.ListCheckRunsOptions) ([]*github.CheckRun, error) {
	return m.ListChecksFunc(ctx, sha, options)
}

func (m *mockPullRequestClient) ListFiles(ctx context.Context, options *github.ListOptions) ([]*github.CommitFile, error) {
	return m.ListFilesFunc(ctx, options)
}

// setupMockPRClient creates a mock PR client with the appropriate behavior for the test case
func setupMockPRClient(checkRuns []*github.CheckRun, listChecksError error, progressiveChecks bool, prFiles []*github.CommitFile) *mockPullRequestClient {
	// Set up a counter for the number of API calls
	callCount := 0

	return &mockPullRequestClient{
		ListChecksFunc: func(ctx context.Context, sha string, opts *github.ListCheckRunsOptions) ([]*github.CheckRun, error) {
			// First call returns error if specified
			if callCount == 0 && listChecksError != nil {
				callCount++
				return nil, listChecksError
			}

			// If progressiveChecks is true, return different results based on call count
			var runs []*github.CheckRun
			if progressiveChecks && len(checkRuns) > callCount {
				runs = []*github.CheckRun{checkRuns[callCount]}
			} else {
				runs = checkRuns
			}

			callCount++
			return runs, nil
		},

		ListFilesFunc: func(ctx context.Context, options *github.ListOptions) ([]*github.CommitFile, error) {
			return prFiles, nil
		},
	}
}

func setupAction(event string, values ...string) (*githubactions.Action, *bytes.Buffer) {
	envMap := map[string]string{
		"GITHUB_EVENT_PATH":   fmt.Sprintf("../../test/events/%s.json", event),
		"GITHUB_STEP_SUMMARY": "/dev/null",
		"GITHUB_REPOSITORY":   "RoryQ/required-checks",
		"GITHUB_RUN_ID":       "12345",
	}

	for keyValue := range slices.Chunk(values, 2) {
		envMap["INPUT_"+keyValue[0]] = keyValue[1]
	}

	getenv := func(key string) string {
		return envMap[key]
	}

	b := new(bytes.Buffer)

	action := githubactions.New(
		githubactions.WithGetenv(getenv),
		githubactions.WithWriter(b),
	)
	return action, b
}
