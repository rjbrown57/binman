package binman

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"runtime"

	"github.com/google/go-github/v50/github"
	"github.com/rjbrown57/binman/pkg/gh"
	"github.com/rjbrown57/binman/pkg/gl"
	log "github.com/rjbrown57/binman/pkg/logging"
	"github.com/rjbrown57/binman/pkg/templating"
	"gitlab.com/gitlab-org/api/client-go"
)

type ReleaseExcludeAction struct {
	r *BinmanRelease
}

func (r *BinmanRelease) AddReleaseExcludeAction() Action {
	return &ReleaseExcludeAction{
		r,
	}
}

func (action *ReleaseExcludeAction) execute() error {
	if len(action.r.ExcludeOs) > 0 {
		for _, os := range action.r.ExcludeOs {
			if os == runtime.GOOS {
				return &ExcludeError{
					RepoName: action.r.Repo,
					Criteria: fmt.Sprintf("OS %s matches an excluded OS", os),
				}
			}
		}
	}
	return nil
}

type ReleaseStatusAction struct {
	r           *BinmanRelease
	releasePath string
}

func (r *BinmanRelease) AddReleaseStatusAction(releasePath string) Action {
	return &ReleaseStatusAction{
		r,
		releasePath,
	}
}

// ReleaseStatusAction verifies whether we have work to do
func (action *ReleaseStatusAction) execute() error {

	action.r.setpublishPath(action.releasePath, action.r.Version)
	_, err := os.Stat(action.r.PublishPath)

	if action.r.watchExposeMetrics {
		var latestLabel string = "true"
		if action.r.QueryType == "releasebytag" {
			latestLabel = "false"
		}
		action.r.metric.WithLabelValues(latestLabel, action.r.SourceIdentifier, action.r.Repo, action.r.Version)
	}

	// If err nil we already have this version, send custom error so gosyncrepo knows to end actions
	// Default to capture any other error cases
	switch err {
	case nil:
		// TODO: Use a pre-defined error variable here
		return &NoUpdateError{
			RepoName: action.r.Repo,
			Version:  action.r.Version,
		}
	default:
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
}

type SetUrlAction struct {
	r *BinmanRelease
}

func (r *BinmanRelease) AddSetUrlAction() Action {
	return &SetUrlAction{
		r,
	}
}

// format a user specified url with release data
func (action *SetUrlAction) execute() error {

	// If user has set an external url use that to grab target
	if action.r.ExternalUrl != "" {
		log.Debugf("User specified url %s", action.r.dlUrl)
		action.r.dlUrl = templating.TemplateString(action.r.ExternalUrl, action.r.getDataMap())
		action.r.assetName = filepath.Base(action.r.dlUrl)
		return nil
	}

	// Attempt to apply templating to os and arch as well. Since we're looking to template these fields we can't rely on
	// getting them directly from action.r.getDataMap() as it may return a templated string instead of what we expect.
	// Instead we rely on setting the data map back to the defaults for the environment to allow the user to template
	// them.
	dataMapWithDefaults := action.r.getDataMap()
	dataMapWithDefaults["os"] = runtime.GOOS
	dataMapWithDefaults["arch"] = runtime.GOARCH
	if action.r.Arch != "" {
		action.r.Arch = templating.TemplateString(action.r.Arch, dataMapWithDefaults)
		log.Debugf("Architecture set to: %s", action.r.Arch)
	}
	if action.r.Os != "" {
		log.Debugf("OS before transition: %s", action.r.Os)
		action.r.Os = templating.TemplateString(action.r.Os, dataMapWithDefaults)
		log.Debugf("OS set to: %s", action.r.Os)
	}

	switch data := action.r.relData.(type) {
	case *github.RepositoryRelease:
		// If the user has requested a specifc asset check for that
		if action.r.ReleaseFileName != "" {
			rFilename := templating.TemplateString(action.r.ReleaseFileName, action.r.getDataMap())
			log.Debugf("Get gh asset by name %s", rFilename)
			action.r.assetName, action.r.dlUrl = gh.GetAssetbyName(rFilename, data.Assets)
		} else {
			// Attempt to find the asset via arch/os
			log.Debugf("Attempt to find github asset for %s", action.r.project)
			action.r.assetName, action.r.dlUrl = selectAsset(action.r.Arch, action.r.Os, action.r.Version, action.r.project, gh.GHGetAssetData(data.Assets))
		}
	case []*gitlab.ReleaseLink:
		// If the user has requested a specifc asset check for that
		if action.r.ReleaseFileName != "" {
			rFilename := templating.TemplateString(action.r.ReleaseFileName, action.r.getDataMap())
			log.Debugf("Get gl asset by name %s", rFilename)
			action.r.assetName, action.r.dlUrl = gl.GetAssetbyName(rFilename, data)
		} else {
			// Attempt to find the asset via arch/os
			log.Debugf("Attempt to find gitlab asset for %s\n", action.r.project)
			action.r.assetName, action.r.dlUrl = selectAsset(action.r.Arch, action.r.Os, action.r.Version, action.r.project, gl.GLGetAssetData(data))
		}
	// TODO should we use a pointer here like the above from better devs than myself?
	case BinmanQueryResponse:
		action.r.dlUrl = data.DlUrl
		action.r.assetName = path.Base(data.DlUrl)
		action.r.Version = data.Version
	}

	// If at this point dlUrl is not set we have an issue
	if action.r.dlUrl == "" {
		return fmt.Errorf("Target release asset not found for %s", action.r.Repo)
	}

	return nil
}

type SetArtifactPathAction struct {
	r           *BinmanRelease
	releasePath string
	binPath     string
}

func (r *BinmanRelease) AddSetArtifactPathAction(releasePath, binPath string) Action {
	return &SetArtifactPathAction{
		r,
		releasePath,
		binPath,
	}
}

func (action *SetArtifactPathAction) execute() error {
	action.r.setArtifactPath(action.releasePath, action.binPath, action.r.assetName)
	// We set cleanupOnFailure to true in case we hit an issue further down the line
	action.r.cleanupOnFailure = true
	// If the binPath string is empty, for example when using `binman get`, don't create the directory
	if action.binPath != "" {
		if err := CreateDirectory(action.binPath); err != nil {
			return err
		}
	}
	err := CreateDirectory(action.r.PublishPath)
	// At this point we have created something during the release process
	return err
}
