package gl

import (
	log "github.com/rjbrown57/binman/pkg/logging"
	"github.com/xanzy/go-gitlab"
)

// Return the Latest Tag for a gitlab repo
func GLGetLatestTag(glClient *gitlab.Client, repo string) string {
	tags, _, err := glClient.Tags.ListTags(repo, &gitlab.ListTagsOptions{OrderBy: gitlab.String("updated"), Sort: gitlab.String("desc")})
	if err == nil {
		return tags[0].Name
	}

	log.Debugf("Error listing tags for %s - %v", tags, err)
	return ""

}

// GLGetTag is used to verify a tag exists
func GLGetTag(glClient *gitlab.Client, repo string, version string) bool {
	_, _, err := glClient.Tags.GetTag(repo, version)
	if err == nil {
		return true
	}

	log.Debugf("Error Getting tag %s for %s - %v", version, repo, err)

	return false
}
