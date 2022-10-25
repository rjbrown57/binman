package binman

import (
	"testing"
)

func TestFormatString(t *testing.T) {

	var templateString = "https://get.helm.sh/helm-{{.version}}-{{.os}}-{{.arch}}.tar.gz"
	var expectedString = "https://get.helm.sh/helm-v0.0.0-linux-amd64.tar.gz"

	m := make(map[string]string)
	m["version"] = "v0.0.0"
	m["os"] = "linux"
	m["arch"] = "amd64"

	testTemplate := formatString(templateString, m)

	if expectedString != testTemplate {
		t.Fatalf("%s was not properly templated to %s - received %s", templateString, expectedString, testTemplate)
	}

	// Verify %s substitution for {{.}} works
	templateString = "https://get.helm.sh/helm-%s-linux-amd64.tar.gz"
	expectedString = "https://get.helm.sh/helm-v0.0.0-linux-amd64.tar.gz"

	testTemplate = formatString(templateString, m)

	if expectedString != testTemplate {
		t.Fatalf("%s was not properly templated to %s - received %s", templateString, expectedString, testTemplate)
	}

	// Verify complex templating
	templateString = `https://releases.hashicorp.com/terraform/{{ trimPrefix "v" .version }}/terraform_{{ trimPrefix "v" .version }}_{{.os}}_{{.arch}}.zip`
	expectedString = "https://releases.hashicorp.com/terraform/0.0.0/terraform_0.0.0_linux_amd64.zip"

	testTemplate = formatString(templateString, m)

	if expectedString != testTemplate {
		t.Fatalf("%s was not properly templated to %s - received %s", templateString, expectedString, testTemplate)
	}

}
