package gl

import (
	log "github.com/rjbrown57/binman/pkg/logging"
	"github.com/xanzy/go-gitlab"
)



func GLGetLatestTag(glClient *gitlab.Client, repo string) string {
	tags, _, err := glClient.Tags.ListTags(repo, &gitlab.ListTagsOptions{OrderBy: gitlab.String("updated"), Sort: gitlab.String("desc")})
	if err != nil {
		log.Debugf("Error listing tags for %s - %v", tags, err)
	}
	return tags[0].Name
}
