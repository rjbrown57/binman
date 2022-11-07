package binman

import (
	"bytes"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	log "github.com/rjbrown57/binman/pkg/logging"
)

// KnownUrlMap contains "projectname/repo" = "downloadurl" for common release assets not hosted on github
var KnownUrlMap = map[string]string{
	"helm/helm":             "https://get.helm.sh/helm-{{.version}}-{{.os}}-{{.arch}}.tar.gz",
	"kubernetes/kubernetes": "https://dl.k8s.io/release/{{.version}}/bin/{{.os}}/{{.arch}}/kubectl",
	"hashicorp/terraform":   `https://releases.hashicorp.com/terraform/{{ trimPrefix "v" .version }}/terraform_{{ trimPrefix "v" .version }}_{{.os}}_{{.arch}}.zip`,
	"hashicorp/vault":       `https://releases.hashicorp.com/vault/{{ trimPrefix "v" .version }}/vault_{{ trimPrefix "v" .version }}_{{.os}}_{{.arch}}.zip`,
}

// Format strings for processing. Currently used by releaseFileName and DlUrl
func formatString(templateString string, dataMap map[string]string) string {

	// For compatability with previous binman versions update %s to {{.}}
	templateString = strings.Replace(templateString, "%s", "{{.version}}", -1)

	// we need an io.Writer to capture the template output
	buf := new(bytes.Buffer)

	// https://github.com/Masterminds/sprig use sprig functions for extra templating functions
	tmpl, err := template.New("stringFormatter").Funcs(sprig.FuncMap()).Parse(templateString)
	if err != nil {
		log.Fatalf("unable to process template for %s", templateString)
	}

	err = tmpl.Execute(buf, dataMap)
	if err != nil {
		log.Fatalf("unable to process template for %s", templateString)
	}

	return buf.String()
}
