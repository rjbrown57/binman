package binman

import (
	"testing"
)

func TestFormatString(t *testing.T) {

	var templateString = "https://get.helm.sh/helm-{{.}}-linux-amd64.tar.gz"
	var tag = "v0.0.0"
	var expectedString = "https://get.helm.sh/helm-v0.0.0-linux-amd64.tar.gz"

	testTemplate := formatString(templateString, tag)

	if expectedString != testTemplate {
		t.Fatalf("%s was not properly templated to %s", templateString, expectedString)
	}

	// Verify %s substitution for {{.}} works
	templateString = "https://get.helm.sh/helm-%s-linux-amd64.tar.gz"
	tag = "v0.0.0"
	expectedString = "https://get.helm.sh/helm-v0.0.0-linux-amd64.tar.gz"

	testTemplate = formatString(templateString, tag)

	if expectedString != testTemplate {
		t.Fatalf("%s was not properly templated to %s", templateString, expectedString)
	}

	// Verify complex templating
	templateString = `https://releases.hashicorp.com/terraform/{{ trimPrefix "v" . }}/terraform_{{ trimPrefix "v" . }}_linux_amd64.zip`
	tag = "v0.0.0"
	expectedString = "https://releases.hashicorp.com/terraform/0.0.0/terraform_0.0.0_linux_amd64.zip"

	testTemplate = formatString(templateString, tag)

	if expectedString != testTemplate {
		t.Fatalf("%s was not properly templated to %s", templateString, expectedString)
	}

}
