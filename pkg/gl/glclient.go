package gl

import (
	"os"

	"net/url"

	log "github.com/rjbrown57/binman/pkg/logging"
	"gitlab.com/gitlab-org/api/client-go"
)

func GetGLClient(baseUrl string, tokenvar string) *gitlab.Client {

	glUrl, err := url.Parse(baseUrl)
	if err != nil {
		log.Fatalf("Unable to parse configured gitlab url %s", baseUrl)
	}

	// if tokenvar is unset we will use anonymous auth
	glToken := os.Getenv(tokenvar)

	gl, err := gitlab.NewClient(glToken, gitlab.WithBaseURL(glUrl.String()))

	if err != nil {
		log.Fatalf("Error getting gitlab client for %s\n", baseUrl)
	}

	return gl
}
