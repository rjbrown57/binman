package gh

import (
	"context"
	"os"

	"github.com/google/go-github/v44/github"
	"golang.org/x/oauth2"
)

// GetGHClient will get a go-github client with auth for api access
func GetGHCLient(tokenvar string) *github.Client {

	// No auth client if user does not supply envvar
	if tokenvar == "none" {
		return github.NewClient(nil)
	}

	ghtoken := os.Getenv(tokenvar)
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: ghtoken},
	)

	tc := oauth2.NewClient(ctx, ts)

	return github.NewClient(tc)
}
