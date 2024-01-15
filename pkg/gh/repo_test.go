package gh

import (
	"errors"
	"testing"
)

func TestGetOr(t *testing.T) {

	var testRepo string = "rjbrown57/binman"
	org, proj := getOR(testRepo)

	if org != "rjbrown57" || proj != "binman" {
		t.Fatalf("repo not split correctly")
	}
}

func TestCheckRepo(t *testing.T) {

	// We use NewBMConfig here to avoid grabbing contextual configs

	var tests = []struct {
		Expected error
		Got      error
		Repo     string
	}{
		{Repo: "rjbrown57/binman", Expected: nil},
		{Repo: "rjbrown57/binman_dne", Expected: &InvalidGhResponse{}},
		{Repo: "rjbrown57badname", Expected: &BadRepoFormat{}},
	}

	for _, test := range tests {
		_, test.Got = CheckRepo(GetGHCLient(defaultGHBaseURL, "none"), test.Repo)

		if test.Expected != nil {
			if errors.Is(test.Expected, test.Got) {
				t.Fatalf("%s Expected \n%s \nGot \n%s", test.Repo, test.Expected, test.Got)
			}
		}
	}

}
