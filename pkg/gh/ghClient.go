package gh

import (
	"context"
	"net/url"
	"os"

	"github.com/google/go-github/v50/github"
	log "github.com/rjbrown57/binman/pkg/logging"
	"golang.org/x/oauth2"
)

// GetGHClient will get a go-github client with auth for api access
func GetGHCLient(baseUrl string, tokenvar string) *github.Client {

	ghUrl, err := url.Parse(baseUrl)
	if err != nil {
		log.Fatalf("Unable to parse configured github url %s", baseUrl)
	}

	// No auth client if user does not supply envvar
	if tokenvar == "none" {
		log.Debugf("Returning github client without auth")
		gh := github.NewClient(nil)
		gh.BaseURL = ghUrl
		return gh
	}

	ghtoken := os.Getenv(tokenvar)

	if len(ghtoken) == 0 {
		log.Fatalf("Specified environment variable %s is empty", tokenvar)
	}

	log.Debugf("Returning github client using %s for auth", tokenvar)

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: ghtoken},
	)

	tc := oauth2.NewClient(ctx, ts)

	gh := github.NewClient(tc)
	gh.BaseURL = ghUrl

	return gh
}
