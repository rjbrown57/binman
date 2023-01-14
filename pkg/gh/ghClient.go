package gh

import (
	"context"
	"os"

	"github.com/google/go-github/v49/github"
	log "github.com/rjbrown57/binman/pkg/logging"
	"golang.org/x/oauth2"
)

// GetGHClient will get a go-github client with auth for api access
func GetGHCLient(tokenvar string) *github.Client {

	// No auth client if user does not supply envvar
	if tokenvar == "none" {
		log.Debugf("Returning github client without auth")
		return github.NewClient(nil)
	}

	log.Debugf("Returning github client using %s for auth", tokenvar)
	ghtoken := os.Getenv(tokenvar)

	if len(ghtoken) == 0 {
		log.Fatalf("Specified environment variable %s is empty", tokenvar)
	}

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: ghtoken},
	)

	tc := oauth2.NewClient(ctx, ts)

	return github.NewClient(tc)
}
