package binman

import (
	"context"
	"fmt"

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
		action.r.createdAtTime = ghd.GetCreatedAt().Unix()
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

	// Get latest tag or Confirm tag exsits
	switch action.r.QueryType {
	case "release":
		log.Debugf("Querying gitlab api for latest release of %s", action.r.Repo)
		action.r.Version, err = gl.GLGetLatestTag(action.glClient, action.r.Repo)
		if err != nil {
			return err
		}
		log.Debugf("Latest release of %s == %s", action.r.Repo, action.r.Version)
	case "releasebytag":
		log.Debugf("Querying gitlab api for tag %s of %s", action.r.Version, action.r.Repo)
		if !gl.GLGetTag(action.glClient, action.r.Repo, action.r.Version) {
			err = fmt.Errorf("Unable to find tag %s for %s", action.r.Version, action.r.Repo)
			return err
		}
	}

	//Fetch release data
	releaseLinks, t := gl.GLGetReleaseAssets(action.glClient, action.r.Repo, action.r.Version)
	action.r.relData = releaseLinks
	action.r.createdAtTime = t

	if action.r.relData == nil || len(releaseLinks) == 0 {
		err = fmt.Errorf("No release data found for %s", action.r.Repo)
	}

	return err
}

type GetBinmanReleaseAction struct {
	r *BinmanRelease
}

func (r *BinmanRelease) AddGetBinmanReleaseAction() Action {
	return &GetBinmanReleaseAction{
		r,
	}
}

func (action *GetBinmanReleaseAction) execute() error {

	q := BinmanQuery{
		Architechure: action.r.Arch,
		Repo:         action.r.Repo,
		Source:       action.r.SourceIdentifier,
		Version:      action.r.Version,
	}

	resp, err := q.SendQuery(action.r.source.URL)
	if err != nil {
		return err
	}

	action.r.relData = resp

	return nil
}
