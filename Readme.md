![intro](logo/binmanlogo.png)

## General

Binman is a tool to sync release assets from github to your local workstation. The main use case is syncing via config to keep the tools you use every day up to date.

Grab the latest release [here](https://github.com/rjbrown57/binman/releases), and let binman grab it for you next time :rocket:

Binman will attempt to find a release asset that matches your OS and architecture and one of the types of files we handle currently. Currently handled file types are "zip", "tar", "binary", "exe". When a proper release asset is found it will be downloaded to your `releasepath` or the default `$HOME/binMan`. The download file will be placed at `$releasepath/repos/${githuborg}/${githubproject}/${version}/`. If the asset is an archive all files will be extracted. Binman will then attempt to find the binary and will link it to `$releasepath/${githubproject}`. Just add the releasepath to your shell PATH var and you are good to go!  

Binman provides many config options to allow you to handle all the manifold release styles on github. Check out the [config options section](#config-options) for details.

## Config Sync 

To run binman you effectively you need a config. Running binman with no arguements and no config will populate the following config to your OS's appropriate [config directory](https://pkg.go.dev/os#UserConfigDir). On linux the config file will be added to `~/.config/binman/config`. Binman allows supplying a configfile from an alternate pathwith the `-c` flag.

Here's an example config file

```yaml
config:
  releasepath:  #path to keep fetched releases. $HOME/binMan is the default
  tokenvar: #environment variable that contains github token
  upx: #Compress binaries with upx
    enabled: false
    args: [] # arrary of args for upx
releases:
  - repo: rjbrown57/binman
    linkname: mybinman  
    downloadonly: false 
    upx: 
      args: [] #["-k","-v"]
```

### Config Options

Top level `config:` options

| key      | Description |
| ----------- | ----------- |
| releasepath | Path to publish files to |
| tokenvar   | github token to use for auth. You can get yourself rate limited if you have a sizeable config. Instructions to [generate a token are here](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token") |
| upx   | config to enable upx shrinking. Details below |

### Release options

These options can be set per release

| key      | Description |
| ----------- | ----------- |
| arch   | target architecture |
| downloadonly   | default `false`. Set to true if you don't want binman to try to extract and link the asset |
| externalurl | see [externalurl support](#external-url-support) |
| linkname | by default binman will create a symlink matching the project name. This can be overidden with linkname set per release |
| releasefilename | in some cases project publish assets that have different names than the github project. For example [cilium-cli](github.com/cilium/cilium-cli) publishs a cli `cilium`. We would set `cilium` here so binman knows what to look for |
| os | target OS  |
| upx | see [upx Config](#upx-config) |
| version | pin to a specific release version |

### External Url Support

binman currently supports fetching version information from github, and then downloading the asset from a seperate url. This comes in handle for projects like [kubernetes](github.com/kubernetes/kubernetes) or [helm](github.com/helm/helm). This allows you to specify a url with the version number replaced with `%s`. The correct version will be added in at download time. 

```
releases:
  - repo: kubernetes/kubernetes
    url: https://kind.sigs.k8s.io/dl/%s/kind-linux-amd64
```

Currently only version number is supported for templating in this style. Expect a [refactor](https://github.com/rjbrown57/binman/issues/19) on this feature soon.

### Upx Config

Binman allows for shrinking of your downloaded binaries via [upx](https://upx.github.io/). Ensure upx is in your path and add the following to your binman config to enable shrinking via UPX.

```yaml
config:
  upx: #Compress binaries with upx
    enabled: false
    args: [] # arrary of args for upx https://linux.die.net/man/1/upx

```

## Direct Repo sync

Binman can also be used to grab a specifc repository with the syntax `binman -r rjbrown57/binman` 

```
binman -r rjbrown57/binman                                                         
INFO[0000] binman sync begin                            
INFO[0000] direct repo download                         
INFO[0000] Downloading https://github.com/rjbrown57/binman/releases/download/v0.0.12/binman_linux_amd64 
INFO[0002] Download https://github.com/rjbrown57/binman/releases/download/v0.0.12/binman_linux_amd64 complete 
INFO[0002] binman finished!          
```

## Using "ghcr.io/rjbrown57/binman:latest" container image

You can use the binman image in a multi-stage build to grab binaries for docker images.

```
FROM ghcr.io/rjbrown57/binman:latest AS binman
RUN binman -r "sigstore/cosign"
FROM ubuntu:latest
COPY --from=binman /cosign-linux-amd64 /usr/bin/cosign
RUN chmod 755 /usr/bin/cosign
```
