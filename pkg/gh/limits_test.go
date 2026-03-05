package gh

import (
	"testing"
)

func TestGetLimits(t *testing.T) {

	client := GetGHCLient(defaultGHBaseURL, "none")
	limits, err := getLimits(client)
	if err != nil {
		t.Errorf("unable to get limits, %s", err)
	}

	if limits.Core.Limit != 60 {
		t.Fatal("Failed to get basic limits")
	}
}
