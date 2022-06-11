# binman

```
                                ___           ___           ___           ___
     _____        ___          /__/\         /__/\         /  /\         /__/\
    /  /::\      /  /\         \  \:\       |  |::\       /  /::\        \  \:\
   /  /:/\:\    /  /:/          \  \:\      |  |:|:\     /  /:/\:\        \  \:\
  /  /:/~/::\  /__/::\      _____\__\:\   __|__|:|\:\   /  /:/~/::\   _____\__\:\
 /__/:/ /:/\:| \__\/\:\__  /__/::::::::\ /__/::::| \:\ /__/:/ /:/\:\ /__/::::::::\
 \  \:\/:/~/:/    \  \:\/\ \  \:\~~\~~\/ \  \:\~~\__\/ \  \:\/:/__\/ \  \:\~~\~~\/
  \  \::/ /:/      \__\::/  \  \:\  ~~~   \  \:\        \  \::/       \  \:\  ~~~
   \  \:\/:/       /__/:/    \  \:\        \  \:\        \  \:\        \  \:\
    \  \::/        \__\/      \  \:\        \  \:\        \  \:\        \  \:\
     \__\/                     \__\/         \__\/         \__\/         \__\/

```

[![Go Report Card](https://goreportcard.com/badge/github.com/rjbrown57/binman)](https://goreportcard.com/report/github.com/rjbrown57/binman)

github Binary manager. Create a yaml file of releases and locations.


```
Github Binary Manager will grab binaries from github for you!

Usage:
  binman [flags]

Flags:
  -c, --config string   path to config file
  -d, --debug           enable debug logging
  -h, --help            help for binman
  -j, --json            enable json style logging
  -r, --repo string     Github repo in format org/repo
```


## Direct download

Supply `-r/--repo` flag to download a single release. This will place the latest release in your current directory. Currently it does not extract tar's but that will be added at a later date.
```
/binman -r anchore/syft
INFO[0000] binman sync begin
INFO[0000] direct repo download
INFO[0000] Querying github api for latest release of anchore/syft
INFO[0000] Getting https://github.com/anchore/syft/releases/download/v0.47.0/syft_0.47.0_darwin_amd64.tar.gz
downloading 100% |███████████████████████████████████████████████████████████████████████████████████████████████████████████████████████████████████████████████████████████████████████████████████████████████████████████| (21/21 MB, 3.026 MB/s)
INFO[0007] binman finished!
```

## Example config file

binman will try it's best to detect either a binary or tar to grab from github that matches your OS and ARCH. Various keys can be set per release to aid in getting/extraction and linking.

```
config:
  #releasepath: /set/path/heretopublishto/ # Default is homeDirectory/binMan 
  tokenvar: GH_TOKEN # GITHUB API TOKEN
defaults:
  checksum: false
  filetype: tar.gz # choose tar.gz or binary
releases:
  - repo: anchore/syft # Easy mode!
  - repo: google/go-containerregistry
    extractfilename: crane # specific file name within a tar.gz
  - repo: GoogleContainerTools/container-structure-test
    releasefilename: container-structure-test-darwin-amd64 # specific release file name
    linkname: cst # Set the link name
  - repo: helm/helm
    url: https://get.helm.sh/helm-%s-linux-amd64.tar.gz
    extractfilename: linux-amd64/helm
  - repo: GoogleContainerTools/container-diff
    downloadonly: true # binman will only download the file. You take care of the rest ;)
    releasefilename: container-diff-darwin-amd64
  - repo: jesseduffield/lazygit
    linkname: lzg
  - repo: jesseduffield/lazydocker
    linkname: lzd

 ```

## Using "ghcr.io/rjbrown57/binman:latest"

You can use the binman image in a multi-stage build to grab binaries for docker images.

```
FROM ghcr.io/rjbrown57/binman:latest AS binman
RUN binman -r "sigstore/cosign"
FROM ubuntu:latest
COPY --from=binman /cosign-linux-amd64 /usr/bin/cosign
RUN chmod 755 /usr/bin/cosign
```

# Flow

* Get releases from GH
* Process releases
  * Try to find a binary or a tar that matches or OS/ARCH. Or the OS/ARCH specified for that specific release.
  * Download releaseFile to ${confg.releasepath}/repos/${org}/${repo}/${tag}/downloadedfileshere
  * Extract notes
  * If file is a tar ball it is extracted to the same directory it was downloaded to
  * Create a symlink from to ${config.releasepath}/${cmdName}
