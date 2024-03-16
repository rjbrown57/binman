package constants

// Common regexes
const TarRegEx = `(\.tar$|\.tar\.gz$|\.tgz$|\.tar\.xz$|\.txz$)`
const ZipRegEx = `(\.zip$)`
const ExeRegex = `.*\.exe$`
const X86RegEx = `(amd64|x86_64)`
const MacOsRx = `(darwin|macos|apple)`
const GzipRegEx = `(\.tar\.gz$|\.tgz$)`
const XzipRegEx = `(\.tar\.xz$|\.txz$)`

// Url defaults
// must have /
const DefaultGHBaseURL = "https://api.github.com/"

// no /
const DefaultGLBaseURL = "https://gitlab.com"

// KnownUrlMap contains "projectname/repo" = "downloadurl" for common release assets not hosted on github
var KnownUrlMap = map[string]string{
	"helm/helm":             "https://get.helm.sh/helm-{{.version}}-{{.os}}-{{.arch}}.tar.gz",
	"kubernetes/kubernetes": "https://dl.k8s.io/release/{{.version}}/bin/{{.os}}/{{.arch}}/kubectl",
	"hashicorp/terraform":   `https://releases.hashicorp.com/terraform/{{ trimPrefix "v" .version }}/terraform_{{ trimPrefix "v" .version }}_{{.os}}_{{.arch}}.zip`,
	"hashicorp/vault":       `https://releases.hashicorp.com/vault/{{ trimPrefix "v" .version }}/vault_{{ trimPrefix "v" .version }}_{{.os}}_{{.arch}}.zip`,
}