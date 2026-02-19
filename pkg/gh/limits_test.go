package gh

import (
	"reflect"
	"testing"

	"github.com/google/go-github/v50/github"
)

func TestGetLimits(t *testing.T) {

	client := GetGHCLient(defaultGHBaseURL, "none")
	_, err := getLimits(client)
	if err != nil {
		t.Errorf("unable to get limits, %s", err)
	}

	// Based on https://github.com/google/go-github/blob/master/github/github_test.go#L468
	if got, want := reflect.TypeFor[github.RateLimits]().NumField(), reflect.TypeFor[github.RateLimits]().NumField(); got != want {
		t.Errorf("len(Client{}.rateLimits) is %v, want %v", got, want)
	}
}
