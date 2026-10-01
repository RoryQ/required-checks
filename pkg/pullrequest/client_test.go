package pullrequest

import (
	"bytes"
	"testing"

	"github.com/sethvargo/go-githubactions"
	"github.com/stretchr/testify/assert"
)

func TestGetRepo(t *testing.T) {
	t.Run("from GITHUB_REPOSITORY env", func(t *testing.T) {
		action := githubactions.New(
			githubactions.WithGetenv(func(key string) string {
				if key == "GITHUB_REPOSITORY" {
					return "owner-name/repo-name"
				}
				return ""
			}),
			githubactions.WithWriter(new(bytes.Buffer)),
		)
		owner, repo := getRepo(action, nil)
		assert.Equal(t, "owner-name", owner)
		assert.Equal(t, "repo-name", repo)
	})

	t.Run("from event payload when env is empty", func(t *testing.T) {
		action := githubactions.New(
			githubactions.WithGetenv(func(key string) string { return "" }),
			githubactions.WithWriter(new(bytes.Buffer)),
		)
		event := map[string]any{
			"repository": map[string]any{
				"full_name": "event-owner/event-repo",
			},
		}
		owner, repo := getRepo(action, event)
		assert.Equal(t, "event-owner", owner)
		assert.Equal(t, "event-repo", repo)
	})

	t.Run("malformed repo string without slash does not panic", func(t *testing.T) {
		action := githubactions.New(
			githubactions.WithGetenv(func(key string) string {
				if key == "GITHUB_REPOSITORY" {
					return "malformed-repo-no-slash"
				}
				return ""
			}),
			githubactions.WithWriter(new(bytes.Buffer)),
		)
		owner, repo := getRepo(action, nil)
		assert.Equal(t, "", owner)
		assert.Equal(t, "", repo)
	})

	t.Run("empty environment and event", func(t *testing.T) {
		action := githubactions.New(
			githubactions.WithGetenv(func(key string) string { return "" }),
			githubactions.WithWriter(new(bytes.Buffer)),
		)
		owner, repo := getRepo(action, map[string]any{})
		assert.Equal(t, "", owner)
		assert.Equal(t, "", repo)
	})
}
