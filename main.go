package main

import (
	"context"
	"net/http"

	"github.com/google/go-github/v92/github"
	"github.com/sethvargo/go-githubactions"
	"golang.org/x/oauth2"

	"github.com/roryq/required-checks/pkg/reqcheck"
	"github.com/roryq/required-checks/pkg/reqcheck/inputs"
)

func run() error {
	ctx := context.Background()
	action := githubactions.New()

	cfg, err := reqcheck.ConfigFromInputs(action)
	if err != nil {
		return err
	}

	var tc *http.Client
	if token := action.GetInput(inputs.Token); token != "" {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		)
		tc = oauth2.NewClient(ctx, ts)
	}

	gh, err := github.NewClient(github.WithHTTPClient(tc))
	if err != nil {
		return err
	}
	return reqcheck.Run(ctx, cfg, action, gh)
}

func main() {
	err := run()
	if err != nil {
		githubactions.Fatalf("%v", err)
	}
}
