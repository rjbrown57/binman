package gh

import (
	"context"

	"github.com/google/go-github/v49/github"
	log "github.com/rjbrown57/binman/pkg/logging"
)

func getLimits(ghClient *github.Client) (*github.RateLimits, error) {

	ctx := context.Background()

	limits, _, err := ghClient.RateLimits(ctx)
	if err != nil {
		log.Debugf("unable to get limits")
		return nil, err
	}

	return limits, nil
}

// ShowLimits will log the current limits values
func ShowLimits(ghClient *github.Client) error {

	limits, err := getLimits(ghClient)
	if err != nil {
		return err
	}

	log.Debugf("Github Rate limit info %s", limits.Core.String())
	return nil
}

// CheckLimits will verify you have not exceeded your quota
func CheckLimits(ghClient *github.Client) error {

	limits, err := getLimits(ghClient)
	if err != nil {
		return err
	}

	if limits.Core.Remaining == 0 {
		log.Fatalf("Github API limits exceeded. %s", limits.Core.String())
	}

	return nil
}
