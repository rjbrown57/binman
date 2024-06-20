package oci

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/rjbrown57/binman/pkg/downloader"
	log "github.com/rjbrown57/binman/pkg/logging"
)

// https://golang.testcontainers.org/features/creating_container/
type regContainer struct {
	testcontainers.Container
	URI string
}

type BuildOciTest struct {
	caseName string
	data     BinmanImageBuild
}

func StartReg(ctx context.Context, t *testing.T) (*regContainer, error) {
	req := testcontainers.ContainerRequest{
		Image:        "registry:2",
		ExposedPorts: []string{"5000/tcp"},
		WaitingFor:   wait.ForHTTP("/"),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})

	if err != nil {
		return nil, err
	}

	ip, err := container.Host(ctx)
	if err != nil {
		return nil, err
	}

	mappedPort, err := container.MappedPort(ctx, "5000")
	if err != nil {
		return nil, err
	}

	uri := fmt.Sprintf("%s:%s", ip, mappedPort.Port())

	// Clean up the container after the test is complete

	r := regContainer{Container: container, URI: uri}

	t.Cleanup(func() {
		if err := r.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate container: %s", err)
		}
	})

	return &r, nil
}

func prepFiles(t *testing.T, testPath string) []string {

	var binPaths []string
	var downloadFiles = []downloader.DlMsg{
		downloader.DlMsg{
			Url: "https://github.com/rjbrown57/binman/releases/download/v0.10.0/binman_linux_amd64",
		},
		downloader.DlMsg{
			Url: "https://github.com/rjbrown57/lp/releases/download/v0.0.4/lp_0.0.4_linux_amd64",
		},
	}

	for _, file := range downloadFiles {

		fileName := filepath.Base(file.Url)
		file.Filepath = fmt.Sprintf("%s/%s", testPath, fileName)
		err := file.DownloadFile()
		if err != nil {
			t.Fatalf("Unable to download test file %s", err)
		}
		log.Tracef("File Dowloaded to %s", file.Filepath)

		binPaths = append(binPaths, file.Filepath)
	}

	return binPaths
}

func TestMakeBinmanImageBuild(t *testing.T) {
	log.ConfigureLog(true, 2)

	var tests = []struct {
		registry        string
		targetImageName string
		baseImage       string
		imagePath       string
		version         string
	}{
		{"localhost:5001", "localhost:5001/rjbrown57/binman:1.0.0", "alpine:latest", "/usr/local/bin/", "1.0.0"},
		{"myreg.myhost.com", "myreg.myhost.com/rjbrown57/binman", "alpine:latest", "/usr/local/bin/", "latest"},
		{"index.docker.io", "rjbrown57/binman", "alpine:latest", "/usr/local/bin/", "latest"},
	}

	for _, test := range tests {
		b, err := MakeBinmanImageBuild(test.targetImageName, test.imagePath, test.baseImage)
		if err != nil {
			t.Fatalf("Failed to make BinmanImageBuild %s", err)
		}
		if b.Registry != test.registry {
			t.Fatalf("Expected registry %s, received %s", test.registry, b.Registry)
		}

		if b.Version != test.version {
			t.Fatalf("Expected version %s, received %s", test.version, b.Version)
		}

	}
}

func TestImageBuild(t *testing.T) {

	// https://docs.github.com/en/actions/learn-github-actions/variables#default-environment-variables
	// Since we are creating containers here this test will not function in github CI.
	if _, isset := os.LookupEnv("CI"); isset {
		t.Skipf("Skipping container based test in CI")
	}

	var err error

	ctx := context.Background()

	r, err := StartReg(ctx, t)
	if err != nil || !r.IsRunning() {
		t.Fatalf("Unable to start Reg %s", err)
	}

	log.ConfigureLog(true, 2)

	log.Debugf("Registry running at %s", r.URI)

	testPath, err := os.MkdirTemp(os.TempDir(), "oci_test")
	if err != nil {
		t.Fatalf("Unable to create test dir")
	}

	defer os.RemoveAll(testPath)

	binPaths := prepFiles(t, testPath)

	var tests = []BuildOciTest{
		{
			caseName: "default",
			data: BinmanImageBuild{
				Assets:       []string{binPaths[0]},
				BaseImage:    "ubuntu:22.04",
				Name:         "rjbrown57/binman",
				Registry:     r.URI,
				ImageBinPath: "/usr/local/bin",
				Version:      "0.0.0",
			},
		},
		{
			caseName: "twobins",
			data: BinmanImageBuild{
				Assets:       binPaths,
				BaseImage:    "ubuntu:22.04",
				Name:         "rjbrown57/binman2",
				Registry:     r.URI,
				ImageBinPath: "/usr/local/bin",
				Version:      "0.0.0",
			},
		}}

	for _, test := range tests {
		err = BuildOciImage(&test.data)
		if err != nil {
			t.Fatalf("Failed to build image %s. Error = %s", "debug", err)
		}

		targetImage := fmt.Sprintf("%s/%s:%s", r.URI, test.data.Name, test.data.Version)
		digest, err := crane.Digest(targetImage)
		if err != nil {
			t.Fatalf("Unable to get digest for %s", targetImage)
		}
		log.Tracef("Digest of %s = %s", targetImage, digest)
	}
}
