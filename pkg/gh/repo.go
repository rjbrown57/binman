package gh

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v50/github"
)

type BadRepoFormat struct {
	repo string
}

func (e *BadRepoFormat) Error() string {
	return fmt.Sprintf("%s should be in the format org/repo", e.repo)
}

type InvalidGhResponse struct {
	resp *github.Response
	err  error
	repo string
}

func (e *InvalidGhResponse) Error() string {
	return fmt.Sprintf("%s could not be verified - error %s - response %d", e.repo, e.err, e.resp.StatusCode)
}

// set project and org vars
func getOR(repo string) (string, string) {
	n := strings.Split(repo, "/")
	return n[0], n[1]
}

func CheckRepo(ghClient *github.Client, repo string) error {

	ctx := context.Background()

	if !strings.Contains(repo, "/") {
		return &BadRepoFormat{repo}
	}

	org, project := getOR(repo)

	_, resp, err := ghClient.Repositories.GetLatestRelease(ctx, org, project)
	if err != nil || resp.StatusCode > 200 {
		return &InvalidGhResponse{resp, err, repo}
	}

	return nil
}
