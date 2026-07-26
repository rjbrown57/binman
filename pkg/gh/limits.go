package gh

import (
	"context"
	"errors"

	"github.com/google/go-github/v50/github"
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

	log.Tracef("Github Rate limit info %s", limits.Core.String())
	return nil
}

// GetRateLimit returns the remaining and total core API quota for the client's
// token. Hitting this endpoint does not count against the primary rate limit,
// so we use it for a cheap, non-fatal pre-flight check.
func GetRateLimit(ghClient *github.Client) (remaining int, limit int, err error) {

	limits, err := getLimits(ghClient)
	if err != nil {
		return 0, 0, err
	}

	return limits.Core.Remaining, limits.Core.Limit, nil
}

// IsRateLimited reports whether err is a GitHub primary or secondary rate limit
// error, so that callers can surface a clear message instead of a generic one.
func IsRateLimited(err error) bool {
	var rlErr *github.RateLimitError
	var abErr *github.AbuseRateLimitError
	return errors.As(err, &rlErr) || errors.As(err, &abErr)
}
