package config

import (
	"os"
	"os/exec"

	binman "github.com/rjbrown57/binman/pkg"
	log "github.com/rjbrown57/binman/pkg/logging"
)

// https://github.com/kubernetes/kubectl/blob/da50ec2b223f5ec08bc34b700411c70b2bcc87fd/pkg/cmd/util/editor/editor.go

func Edit(config string) {

	editorPath := getEditor()
	cPath := binman.SetBaseConfig(config)

	log.Infof("opening %s", editorPath)

	cmd := exec.Command(editorPath, cPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		log.Fatalf("Error executing %s %s %s ---", editorPath, cPath, err)
	}

}

func getEditor() string {
	e, err := exec.LookPath(os.Getenv("EDITOR"))

	if err != nil {
		log.Fatalf("Unable to find editor %s", err)
	}

	return e
}
