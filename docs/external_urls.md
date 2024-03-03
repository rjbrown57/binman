## External Url Support

binman currently supports fetching version information from github, and then downloading the asset from a separate url. Templating via go templates and [sprig](https://masterminds.github.io/sprig/) can be performed on the url to allow substitution of the fetched tag.

The following values are provided that are commonly used with external urls. See [string templating](../docs/templating.md) for a full list.

* os
* arch
* version

```yaml
releases:
  - repo: kubernetes/kubernetes # a basic example
    url: "https://dl.k8s.io/release/{{.version}}/bin/{{.os}}/{{.arch}}/kubectl",
  - repo: hashicorp/terraform # a sprig example
    url: https://releases.hashicorp.com/terraform/{{ trimPrefix "v" .version }}/terraform_{{ trimPrefix "v" .version }}_{{.os}}_{{.arch}}.zip`, 

```

 For convenience a list of "known" repositories is kept with the templating all figured out for you. Just leave the url field blank for these and binman will take care of it.

 Current "known" repos are:

* kubernetes/kubernetes
  * Please note this is currently hardcoded to fetch kubectl.
* hashicorp/terraform
* hashicorp/vault
