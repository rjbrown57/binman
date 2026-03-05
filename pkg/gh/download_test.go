package gh

import (
	"context"
	"net/http"
	"os"
	"testing"
)

func TestGHAssetDownload(t *testing.T) {

	if os.Getenv("GH_TOKEN") == "" {
		t.SkipNow()
	}
	client := GetGHCLient(defaultGHBaseURL, "GH_TOKEN")

	data, resp, err := client.Repositories.GetLatestRelease(context.Background(), "rjbrown57", "binman-private")
	if err != nil {
		t.Fatalf("failed to get repo data %v %s", resp.StatusCode, err)
	}

	out, url, err := client.Repositories.DownloadReleaseAsset(context.TODO(), "rjbrown57", "binman", *data.Assets[0].ID, http.DefaultClient)
	if err != nil {
		t.Fatalf("Failed to download asset %s", err)
	}

	out.Close()
	t.Fatalf("%s", url)

}
