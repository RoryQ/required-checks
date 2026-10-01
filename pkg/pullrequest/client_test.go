package pullrequest

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/google/go-github/v92/github"
	"github.com/sethvargo/go-githubactions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestNewClient(t *testing.T) {
	t.Run("pull_request event", func(t *testing.T) {
		eventData, err := json.Marshal(map[string]any{
			"pull_request": map[string]any{
				"number": 42.0,
			},
			"repository": map[string]any{
				"full_name": "owner/repo",
			},
		})
		require.NoError(t, err)

		tmpFile, err := os.CreateTemp("", "event-*.json")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())
		_, err = tmpFile.Write(eventData)
		require.NoError(t, err)
		tmpFile.Close()

		t.Setenv("GITHUB_EVENT_NAME", "pull_request")
		t.Setenv("GITHUB_EVENT_PATH", tmpFile.Name())
		t.Setenv("GITHUB_REPOSITORY", "owner/repo")

		gh, err := github.NewClient()
		require.NoError(t, err)
		action := githubactions.New(githubactions.WithWriter(new(bytes.Buffer)))
		client, err := NewClient(action, gh)
		require.NoError(t, err)
		assert.True(t, client.Number.Valid)
		assert.Equal(t, 42, client.Number.V)
		assert.Equal(t, "owner", client.Owner)
		assert.Equal(t, "repo", client.Repo)
	})

	t.Run("push event without PR number", func(t *testing.T) {
		eventData, err := json.Marshal(map[string]any{
			"ref": "refs/heads/main",
			"repository": map[string]any{
				"full_name": "owner/repo",
			},
		})
		require.NoError(t, err)

		tmpFile, err := os.CreateTemp("", "event-*.json")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())
		_, err = tmpFile.Write(eventData)
		require.NoError(t, err)
		tmpFile.Close()

		t.Setenv("GITHUB_EVENT_NAME", "push")
		t.Setenv("GITHUB_EVENT_PATH", tmpFile.Name())
		t.Setenv("GITHUB_REPOSITORY", "owner/repo")

		gh, err := github.NewClient()
		require.NoError(t, err)
		action := githubactions.New(githubactions.WithWriter(new(bytes.Buffer)))
		client, err := NewClient(action, gh)
		require.NoError(t, err)
		assert.False(t, client.Number.Valid)
		assert.Equal(t, "owner", client.Owner)
		assert.Equal(t, "repo", client.Repo)
	})

	t.Run("merge_group event", func(t *testing.T) {
		eventData, err := json.Marshal(map[string]any{
			"merge_group": map[string]any{
				"head_sha": "abc1234",
			},
			"repository": map[string]any{
				"full_name": "owner/repo",
			},
		})
		require.NoError(t, err)

		tmpFile, err := os.CreateTemp("", "event-*.json")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())
		_, err = tmpFile.Write(eventData)
		require.NoError(t, err)
		tmpFile.Close()

		t.Setenv("GITHUB_EVENT_NAME", "merge_group")
		t.Setenv("GITHUB_EVENT_PATH", tmpFile.Name())
		t.Setenv("GITHUB_REPOSITORY", "owner/repo")

		gh, err := github.NewClient()
		require.NoError(t, err)
		action := githubactions.New(githubactions.WithWriter(new(bytes.Buffer)))
		client, err := NewClient(action, gh)
		require.NoError(t, err)
		assert.False(t, client.Number.Valid)
		assert.Equal(t, "owner", client.Owner)
		assert.Equal(t, "repo", client.Repo)
	})
}
