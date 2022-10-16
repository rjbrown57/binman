package gh

import (
	"context"
	"fmt"
	"os"
	"testing"
)

// This is the default value for TokenExpiration for an anonymous user
const anonymousTokenExpiration = "0001-01-01 00:00:00 +0000 UTC"

// TestGetGHClientAnonymous will verify we get an anonymous client when expected
func TestGetGHClientAnonymous(t *testing.T) {
	client := GetGHCLient("none")
	_, response, _ := client.APIMeta(context.Background())

	expirationValue := fmt.Sprintf("%v", response.TokenExpiration)
	if expirationValue != anonymousTokenExpiration {
		t.Fatalf("Expected anonymous authentication to be used. Expected %s but found %s", anonymousTokenExpiration, expirationValue)
	}
}

// TestGetGHClientWithAuth will verify we get an anonymous client when expected. This test will do nothing if token is not present
func TestGetClientWithAuth(t *testing.T) {

	if os.Getenv("GH_TOKEN") != "" {
		client := GetGHCLient("GH_TOKEN")
		_, response, _ := client.APIMeta(context.Background())

		expirationValue := fmt.Sprintf("%v", response.TokenExpiration)
		if expirationValue == anonymousTokenExpiration {
			t.Fatalf("Expected authentication to be used. Expected %s should not = %s", anonymousTokenExpiration, expirationValue)
		}
	}

}
