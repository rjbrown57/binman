package gl

import (
	"os"

	log "github.com/rjbrown57/binman/pkg/logging"
	"github.com/xanzy/go-gitlab"
)

func GetGLClient(baseUrl string, tokenvar string) *gitlab.Client {

	glToken := os.Getenv(tokenvar)

	if len(glToken) == 0 {
		log.Fatalf("Specified environment variable %s is empty", tokenvar)
	}
	//gl, err := gitlab.NewClient(glToken,
	gl, err := gitlab.NewClient(glToken,

		// something is wrong with the default gitlab url
		gitlab.WithBaseURL("https://gitlab.com"),
	)

	if err != nil {
		log.Fatalf("Error getting gitlab client for %s\n", baseUrl)
	}

	return gl
}
