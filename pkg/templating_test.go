package binman

import (
	"testing"
)

func TestFormatString(t *testing.T) {

	var tests = []struct {
		templateString string
		expectedString string
	}{
		{"https://get.helm.sh/helm-{{.version}}-{{.os}}-{{.arch}}.tar.gz", "https://get.helm.sh/helm-v0.0.0-linux-amd64.tar.gz"},
		{"https://get.helm.sh/helm-%s-linux-amd64.tar.gz", "https://get.helm.sh/helm-v0.0.0-linux-amd64.tar.gz"},
		{`https://releases.hashicorp.com/terraform/{{ trimPrefix "v" .version }}/terraform_{{ trimPrefix "v" .version }}_{{.os}}_{{.arch}}.zip`, "https://releases.hashicorp.com/terraform/0.0.0/terraform_0.0.0_linux_amd64.zip"},
	}

	m := make(map[string]string)
	m["version"] = "v0.0.0"
	m["os"] = "linux"
	m["arch"] = "amd64"

	for _, test := range tests {
		if retval := formatString(test.templateString, m); retval != test.expectedString {
			t.Fatalf("%s should template to %s got %s", test.templateString, test.expectedString, retval)
		}
	}
}
