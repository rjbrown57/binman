package binman

import (
	"context"

	"github.com/google/go-github/v48/github"
	log "github.com/rjbrown57/binman/pkg/logging"
)

type GetGHLatestReleaseAction struct {
	r        *BinmanRelease
	ghClient *github.Client
}

func (r *BinmanRelease) AddGetGHLatestReleaseAction(ghClient *github.Client) Action {
	return &GetGHLatestReleaseAction{
		r,
		ghClient,
	}
}

func (action *GetGHLatestReleaseAction) execute() error {

	var err error

	ctx := context.Background()

	log.Debugf("Querying github api for latest release of %s", action.r.Repo)
	// https://docs.github.com/en/rest/releases/releases#get-the-latest-release
	action.r.githubData, _, err = action.ghClient.Repositories.GetLatestRelease(ctx, action.r.org, action.r.project)

	return err
}

type GetGHReleaseByTagsAction struct {
	r        *BinmanRelease
	ghClient *github.Client
}

func (r *BinmanRelease) AddGetGHReleaseByTagsAction(ghClient *github.Client) Action {
	return &GetGHReleaseByTagsAction{
		r,
		ghClient,
	}
}

func (action *GetGHReleaseByTagsAction) execute() error {

	var err error

	ctx := context.Background()

	log.Debugf("Querying github api for %s release of %s", action.r.Version, action.r.Repo)
	// https://docs.github.com/en/rest/releases/releases#get-the-latest-release
	action.r.githubData, _, err = action.ghClient.Repositories.GetReleaseByTag(ctx, action.r.org, action.r.project, action.r.Version)

	return err
}
