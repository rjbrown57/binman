package binman

import (
	"context"

	"github.com/google/go-github/v50/github"
	"github.com/rjbrown57/binman/pkg/gl"
	log "github.com/rjbrown57/binman/pkg/logging"
	"github.com/xanzy/go-gitlab"
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
	var ghd *github.RepositoryRelease

	ctx := context.Background()

	switch action.r.QueryType {
	case "release":
		log.Debugf("Querying github api for latest release of %s", action.r.Repo)
		// https://docs.github.com/en/rest/releases/releases#get-the-latest-release
		ghd, _, err = action.ghClient.Repositories.GetLatestRelease(ctx, action.r.org, action.r.project)
	case "releasebytag":
		ghd, _, err = action.ghClient.Repositories.GetReleaseByTag(ctx, action.r.org, action.r.project, action.r.Version)
	}

	if err == nil {
		action.r.Version = ghd.GetTagName()
		action.r.relNotes = ghd.GetBody()
	}

	action.r.relData = ghd

	return err
}

type GetGLReleaseAction struct {
	r        *BinmanRelease
	glClient *gitlab.Client
}

func (r *BinmanRelease) AddGetGLReleaseAction(glClient *gitlab.Client) Action {
	return &GetGLReleaseAction{
		r,
		glClient,
	}
}

func (action *GetGLReleaseAction) execute() error {

	var err error

	switch action.r.QueryType {
	case "release":
		log.Debugf("Querying github api for latest release of %s", action.r.Repo)
		action.r.Version = gl.GLGetLatestTag(action.glClient, action.r.Repo)
	}

	action.r.relData = gl.GLGetReleaseAssets(action.glClient, action.r.Repo, action.r.Version)

	return err
}
