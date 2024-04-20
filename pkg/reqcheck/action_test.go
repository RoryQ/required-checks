package reqcheck

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/sethvargo/go-githubactions"
)

func TestRun(t *testing.T) {
}

func setupAction(input string) (*githubactions.Action, *bytes.Buffer) {
	envMap := map[string]string{
		"GITHUB_EVENT_PATH":   fmt.Sprintf("../../test/events/%s.json", input),
		"GITHUB_STEP_SUMMARY": "/dev/null",
		"GITHUB_REPOSITORY":   "RoryQ/required-checks",
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
