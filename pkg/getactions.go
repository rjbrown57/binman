package binman

import (
	"context"

	"github.com/google/go-github/v48/github"
	log "github.com/rjbrown57/binman/pkg/logging"
)

type GetGHReleaseAction struct {
	r        *BinmanRelease
	ghClient *github.Client
}

func (r *BinmanRelease) AddGetGHReleaseAction(ghClient *github.Client) Action {
	return &GetGHReleaseAction{
		r,
		ghClient,
	}
}

func (action *GetGHReleaseAction) execute() error {

	var err error

	ctx := context.Background()

	if action.r.Version == "" {
		log.Debugf("Querying github api for latest release of %s", action.r.Repo)
		// https://docs.github.com/en/rest/releases/releases#get-the-latest-release
		action.r.githubData, _, err = action.ghClient.Repositories.GetLatestRelease(ctx, action.r.org, action.r.project)
	} else {
		log.Debugf("Querying github api for %s release of %s", action.r.Version, action.r.Repo)
		// https://docs.github.com/en/rest/releases/releases#get-the-latest-release
		action.r.githubData, _, err = action.ghClient.Repositories.GetReleaseByTag(ctx, action.r.org, action.r.project, action.r.Version)
	}

	return err
}
