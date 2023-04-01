package binman

import (
	"context"

	"github.com/google/go-github/v50/github"
	"github.com/rjbrown57/binman/pkg/gh"
	"github.com/rjbrown57/binman/pkg/gl"
	log "github.com/rjbrown57/binman/pkg/logging"
	"github.com/xanzy/go-gitlab"
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
		// this debug message is wrong, this var is unset :) if we are in this else block
		log.Debugf("Attempt to find asset %s", action.r.ReleaseFileName)
		action.r.assetName, action.r.dlUrl = gh.FindAsset(action.r.Arch, action.r.Os, action.r.Version, action.r.project, ghd.Assets)
	}

	return err
}

type GetGLLatestReleaseAction struct {
	r        *BinmanRelease
	glClient *gitlab.Client
}

func (r *BinmanRelease) AddGetGLLatestReleaseAction(glClient *gitlab.Client) Action {
	return &GetGLLatestReleaseAction{
		r,
		glClient,
	}
}

func (action *GetGLLatestReleaseAction) execute() error {

	var err error

	log.Debugf("Querying gitlab api for latest release of %s", action.r.Repo)
	// https://docs.github.com/en/rest/releases/releases#get-the-latest-release
	action.r.Version = gl.GLGetLatestTag(action.glClient, action.r.Repo)

	gld := gl.GLGetReleaseAssets(action.glClient, action.r.Repo, action.r.Version)

	// If the user has requested a specifc asset check for that
	if action.r.ReleaseFileName != "" {
		rFilename := formatString(action.r.ReleaseFileName, action.r.getDataMap())
		log.Debugf("Get asset by name %s", rFilename)
		action.r.assetName, action.r.dlUrl = gl.GetAssetbyName(rFilename, gld)
	} else {
		// Attempt to find the asset via arch/os
		log.Debugf("Attempt to select asset for %s\n", action.r.project)
		action.r.assetName, action.r.dlUrl = gl.FindAsset(action.r.Arch, action.r.Os, action.r.Version, action.r.project, gld)
	}

	return err
}

type GetGLReleaseByTagsAction struct {
	r        *BinmanRelease
	glClient *gitlab.Client
}

func (r *BinmanRelease) AddGetGLReleaseByTagsAction(glClient *gitlab.Client) Action {
	return &GetGLReleaseByTagsAction{
		r,
		glClient,
	}
}

func (action *GetGLReleaseByTagsAction) execute() error {

	var err error

	gld := gl.GLGetReleaseAssets(action.glClient, action.r.Repo, action.r.Version)

	// If the user has requested a specifc asset check for that
	if action.r.ReleaseFileName != "" {
		rFilename := formatString(action.r.ReleaseFileName, action.r.getDataMap())
		log.Debugf("Get asset by name %s", rFilename)
		action.r.assetName, action.r.dlUrl = gl.GetAssetbyName(rFilename, gld)
	} else {
		// Attempt to find the asset via arch/os
		log.Debugf("Attempt to select asset for %s\n", action.r.project)
		action.r.assetName, action.r.dlUrl = gl.FindAsset(action.r.Arch, action.r.Os, action.r.Version, action.r.project, gld)
	}

	return err
}
