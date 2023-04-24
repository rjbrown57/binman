![intro](logo/binmanlogo.png)

## General

Binman is a tool to sync release assets from github or gitlab to your local workstation. The main use case is syncing via config to keep the tools you use every day up to date.

![demo](examples/demo.gif)

Grab the latest release [here](https://github.com/rjbrown57/binman/releases), and let binman grab it for you next time :rocket:

Binman will attempt to find a release asset that matches your OS and architecture and one of the types of files we handle currently. Currently handled file types are "zip", "tar", "binary", "exe". 

Just add the releasepath to your shell PATH var and you are good to go!  

## Config Sync

To run binman effectively you need a config. 

Running binman with no arguements, and no config will populate the defualt config to your OS's appropriate [config directory](https://pkg.go.dev/os#UserConfigDir). On linux the config file will be added to `~/.config/binman/config`. 

Binman also allows supplying a configfile from an alternate path with the `-c` flag or by the "BINMAN_CONFIG" environment variable.

Here's an example config file

```yaml
config:
  releasepath:  #path to keep fetched releases. $HOME/binMan is the default
  cleanup: true # remove downloaded archive
  maxdownloads: 1 # number of concurrent downloads allowed. Default is 3
  upx: #Compress binaries with upx
    enabled: false
    args: [] # arrary of args for upx
  sources:
   - name: gitlab.com
     #tokenvar: GL_TOKEN # environment variable that contains gitlab token
     apitype: gitlab
   - name: github.com
     #tokenvar: GH_TOKEN # environment variable that contains github token
     apitype: github
releases:
  - repo: rjbrown57/binman
    linkname: mybinman  
    downloadonly: false 
    cleanup: true
    upx: 
      args: [] #["-k","-v"]
  # syncing from gitlab
  #- repo: gitlab.com/gitlab-org/cli
```

Binman can also run with a "contextual" config file. If a directory contains a file ".binMan.yaml" this will be merged with your main config. Check a config into git projects and easily fetch required dependencies from github.

### Config Options

Top level `config:` options

| key      | Description |
| ----------- | ----------- |
| cleanup   | Remove .zip/.tar files after we have extracted something. Useful in container builds / CI |
| maxdownloads | number of concurrent downloads to allow. Default is number of releases |
| releasepath | Path to publish files to |
| tokenvar   | github token to use for auth. You can get yourself rate limited if you have a sizeable config. Instructions to [generate a token are here](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token"). This config.tokenvar is left for compatability and can also be set in config.sources for github.com |
| upx   | config to enable upx shrinking. Details below |

### Config sources

By default binman configures two sources `github.com` and `gitlab.com` without authentication. Currently the only supported apitypes are `github` and `gitlab`.  You can supply config to use your internal github or gitlab instances like the below example. Downloads do not currently have authentication, expect this in a future release!

```
config:
  releasepath: # path to keep fetched releases. $HOME/binMan is the default
  maxdownloads: 1
  sources:
   - name: gitlab.com
     tokenvar: GL_TOKEN
     apitype: gitlab
   - name: github.com
     tokenvar: GH_TOKEN
     apitype: github
   - name: myprvate.github.com
     tokenvar: GH_TOKEN
     apitype: github
   - name: myprivate.gitlab.com
     tokenvar: GL_TOKEN
     apitype: gitlab
releases:
  - repo: rjbrown57/binman # by default github will be the source
  - repo: myprivate.github.com/myorg/myproject # source can be supplied in the repo key
  - repo: mygitlaborg/mygitlabproject 
    source: myprivate.gitlab.com # source can also be supplied via the source key. source must match the name field of configured sources.
```

### Release options

These options can be set per release

| key      | Description |
| ----------- | ----------- |
| arch   | target architecture |
| cleanup   | Remove .zip/.tar files after we have extracted something. Useful in container builds / CI |
| downloadonly   | default `false`. Set to true if you don't want binman to try to extract and link the asset |
| externalurl | see [externalurl support](#external-url-support) |
| linkname | by default binman will create a symlink matching the project name. This can be overidden with linkname set per release |
| os | target OS  |
| releasefilename | in some cases project publish assets that have different names than the github project. For example [cilium-cli](github.com/cilium/cilium-cli) publishs a cli `cilium`. We would set `cilium` here so binman knows what to look for |
| releasepath | Alternate releasepath from what is set in the main config |
| source | git source to get release from. By default set to "github.com". Must match the name key of a configured source. See [config-sources](#config-sources)
| upx | see [upx Config](#upx-config) |
| version | pin to a specific release version |
| postcommands | see [post commands](#post-commands)|
| postonly | only run [post commands](#post-commands) after we have checked for new versions. This allows binman to trigger apt/yum/brew or something like that|

## Binman Config subcommand

The `binman config` subcommand can be used for operations related to your binman config file. Use`-c` or `$BINMAN_CONFIG`  for a non standard config path.

### get
to view your config run `binman config get`

### edit
to edit your config run `binman config edit`. This command will make use of whatever editor your $EDITOR var is pointed at.

### add
To add a new repo to your config you can run `binman config add anchore/syft`. This will add `repo: anchore/syft` to our config file in the releases section. If further configuration is required do so with `binman config edit`

## External Url Support

binman currently supports fetching version information from github, and then downloading the asset from a seperate url. Templating via go templates and [sprig](https://masterminds.github.io/sprig/) can be performed on the url to allow substitution of the fetched tag.

The following values are provided that are commonly used with external urls. See [string templating](#string-templating) for a full list.

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
  * Please note this is currently harcoded to fetch kubectl. If you want a different binary set additional `repo: kubernetes/kubernetes` and specify the url field for each additional binary.
* helm/helm
* hashicorp/terraform
* hashicorp/vault

## Post Commands

binman supports executing arbitrary os commands after it has fetched a release artifact. The templating detailed in [string templating](#string-templating) is available to postcommand args. A simple example is to copy the file to a new location.

```yaml
releases:
  - repo: rjbrown57/binextractor
      releasefilename: binextractor_0.0.1-alpha_linux_amd64
      downloadonly: true
      postcommands:
      - command: cp
        args: ["{{ .artifactpath }}","/tmp/binextractor"]
```

A more complex example would be to do a docker build.

```yaml
releases:
  - repo: rjbrown57/binman
    postcommands:
    - command: docker
      args: ["build","-t","{{ .project }}","--build-arg","VERSION={{ .version }}","--build-arg","FILENAME={{ .filename }}","/home/myuser/binMan/repos/{{ .org }}/{{ .project }}/"]
```

For this to work you must place a docker file at `~/binMan/repos/rjbrown57/binman/Dockerfile`. An example of the Dockerfile is

```Dockerfile
FROM ubuntu:22.04
ARG VERSION
ARG FILENAME
COPY $VERSION/$FILENAME /usr/local/bin/$FILENAME
```

These are just a pair of possible postcommands. See what trouble you can get yourself into :rocket:

### Upx Config

Binman allows for shrinking of your downloaded binaries via [upx](https://upx.github.io/). Ensure upx is in your path and add the following to your binman config to enable shrinking via UPX.

```yaml
config:
  upx: #Compress binaries with upx
    enabled: false
    args: [] # arrary of args for upx https://linux.die.net/man/1/upx

```

## String Templating

Binman supports templating via go templates and [sprig](https://masterminds.github.io/sprig/).

Templating is available on the following fields

* url
* releasefilename
* postcommands args

The following values are provided
| key | notes |
| ----------- | ----------- |
| os | the configured os. Usually the os of your workstation |
| arch | the configured architecture. Usually the arch of your workstation
| version | the asset version we have fetched from github |
| project | the github project name |
| org | the github org name |
| artifactpath | the full path to the final extracted release artifact. * |
| link | the full path to link binman creates. * |
| filename | just the file name of the final release artifact. * |

\* these values are only available to args in postcommands actions.

## Binman watch
If you want to run binman continously, configure the watch portion of your config file and invoke `binman watch`. This will also expose `/metrics` and `/healthz` on port 9091 by default. An example config is below.

```
config:
    sources:
        - name: github.com
          tokenvar: GH_TOKEN
          url: https://api.github.com/
          apitype: github
        - name: gitlab.com
          url: https://gitlab.com
          apitype: gitlab
    watch:
      sync: true # Whether to download assets or not. If you only want metrics set to false
      frequency: 60 # seconds between iteration, default is 60
      port: 9091 # Port to expose healthz and metrics endpoints, default is 9091
releases:
- repo: rjbrown57/binman
```

metrics are exposed in the format `binman_release{latest="true",repo="rjbrown57/binman",version="v0.8.0"} 0`. Keep in mind github api limits when configuring how often binman checks for new assets. An example helm chart is provided [here for running in k8s](charts/binman/) 

## Direct Repo sync

Binman can be used to grab a specifc repository with the syntax `binman get rjbrown57/binman`

## Using "ghcr.io/rjbrown57/binman:latest" container image

You can use the binman image in a multi-stage build to grab binaries for docker images.

```Dockerfile
FROM ghcr.io/rjbrown57/binman:latest AS binman
RUN binman get "sigstore/cosign"
FROM ubuntu:latest
COPY --from=binman /cosign-linux-amd64 /usr/bin/cosign
RUN chmod 755 /usr/bin/cosign
```

Or use with a config file
```Dockerfile
FROM ghcr.io/rjbrown57/binman:latest AS binman
ADD myconfig.yaml examples/basicExample.yaml
RUN binman -c examples/basicExample.yaml
FROM ubuntu:latest
COPY --from=binman /root/binMan/ /usr/local/bin/
```
