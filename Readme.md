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
```

## Example config file


```
config:
  releasepath: /home/lookfar/binMan/
  tokenvar: GH_TOKEN # GITHUB API environment variable
defaults:
  checksum: false
  filetype: tar.gz # file ending.
  version: latest # Ignored for now
  prerelease: false
releases:
  - repo: anchore/syft
  - repo: anchore/grype
 ```

## Types of releases currently handled

tarball where binary is at root of tarball and binary matches repo name.

```
releases:
  - repo: anchore/syft
```

A published binary. releasefilename is set to the exact name of the published binary to grab.
```
releases:
  - repo: GoogleContainerTools/container-structure-test
    releasefilename: container-structure-test-linux-amd64 

```

A tarball with a filename different than the repo and a specified Arch to look for.
```
  - repo: google/go-containerregistry
    arch: x86_64
    filename: crane
```

A tarball with a binary not located at the root path

```
 - repo: moby/buildkit
    arch: linux-amd64
    filename: bin/buildctl
```

# Flow

* Get releases from GH
* Process releases
  * Download releaseFile to ${confg.releasepath}/repos/${org}/${repo}/${tag}/downloadedfileshere
  * If file is a tar ball it is extracted to the same directory it was downloaded to
  * Create a symlink from to ${config.releasepath}/${cmdName}
