package binman

import (
	"context"

	"github.com/google/go-github/v50/github"
	"github.com/rjbrown57/binman/pkg/gh"
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
	ghd, _, err := action.ghClient.Repositories.GetLatestRelease(ctx, action.r.org, action.r.project)
	if err == nil {
		action.r.Version = ghd.GetTagName()
		action.r.relNotes = ghd.GetBody()
	}

	// If the user has requested a specifc asset check for that
	if action.r.ReleaseFileName != "" {
		rFilename := formatString(action.r.ReleaseFileName, action.r.getDataMap())
		log.Debugf("Get asset by name %s", rFilename)
		action.r.assetName, action.r.dlUrl = gh.GetAssetbyName(rFilename, ghd.Assets)
	} else {
		// Attempt to find the asset via arch/os
		log.Debugf("Attempt to find asset %s", action.r.ReleaseFileName)
		action.r.assetName, action.r.dlUrl = gh.FindAsset(action.r.Arch, action.r.Os, action.r.Version, action.r.project, ghd.Assets)
	}

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

// Verify specified version exists
func (action *GetGHReleaseByTagsAction) execute() error {

	var err error

	ctx := context.Background()

	log.Debugf("Querying github api for %s release of %s", action.r.Version, action.r.Repo)
	// https://docs.github.com/en/rest/releases/releases#get-the-latest-release
	ghd, _, err := action.ghClient.Repositories.GetReleaseByTag(ctx, action.r.org, action.r.project, action.r.Version)
	action.r.relNotes = ghd.GetBody()

	// If the user has requested a specifc asset check for that
	if action.r.ReleaseFileName != "" {
		rFilename := formatString(action.r.ReleaseFileName, action.r.getDataMap())
		log.Debugf("Get asset by name %s", rFilename)
		action.r.assetName, action.r.dlUrl = gh.GetAssetbyName(rFilename, ghd.Assets)
	} else {
		// Attempt to find the asset via arch/os
		log.Debugf("Attempt to find asset %s", action.r.ReleaseFileName)
		action.r.assetName, action.r.dlUrl = gh.FindAsset(action.r.Arch, action.r.Os, action.r.Version, action.r.project, ghd.Assets)
	}

	return err
}
