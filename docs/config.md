# Config Options

Top level `config:` options

| key      | Description |
| ----------- | ----------- |
| cleanup   | Remove .zip/.tar files after we have extracted something. Useful in container builds / CI |
| maxdownloads | number of concurrent downloads to allow. Default is number of releases |
| releasepath | Path to publish files to |
| binpath | Path to directory where symlinks to binaries will be created, defaults to releasepath |
| tokenvar   | github token to use for auth. You can get yourself rate limited if you have a sizeable config. Instructions to [generate a token are here](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token"). This config.tokenvar is left for compatibility and can also be set in config.sources for github.com |
| upx   | config to enable upx shrinking. Details below |

## Config sources

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
   - name: myprivate.github.com
     tokenvar: GH_TOKEN
     apitype: github
   - name: myprivate.gitlab.com
     tokenvar: GL_TOKEN
     apitype: gitlab
   - name: myprivatebinman.mycompany.com
     apitype: binman
releases:
  - repo: rjbrown57/binman # by default github will be the source
  - repo: myprivate.github.com/myorg/myproject # source can be supplied in the repo key
  - repo: mygitlaborg/mygitlabproject
    source: myprivate.gitlab.com # source can also be supplied via the source key. source must match the name field of configured sources.
```

## Release options

These options can be set per release

| key      | Description |
| ----------- | ----------- |
| arch   | target architecture |
| cleanup   | Remove .zip/.tar files after we have extracted something. Useful in container builds / CI |
| downloadonly   | default `false`. Set to true if you don't want binman to try to extract and link the asset |
| externalurl | see [externalurl support](../docs/external_urls.md) |
| linkname | by default binman will create a symlink matching the project name. This can be overridden with linkname set per release |
| os | target OS  |
| releasefilename | in some cases project publish assets that have different names than the github project. For example [cilium-cli](github.com/cilium/cilium-cli) publishes a cli `cilium`. We would set `cilium` here so binman knows what to look for |
| releasepath | Alternate releasepath from what is set in the main config |
| source | git source to get release from. By default set to "github.com". Must match the name key of a configured source. See [config-sources](#config-sources)
| upx | see [upx Config](../docs/upx.md) |
| version | pin to a specific release version |
| postcommands | see [post commands](../docs/postcommands.md)|
| postonly | only run [post commands](../docs/postcommands.md) after we have checked for new versions. This allows binman to trigger apt/yum/brew or something like that |

## Binman Config subcommand

The `binman config` subcommand can be used for operations related to your binman config file. Use`-c` or `$BINMAN_CONFIG`  for a non standard config path.

### get
to view your config run `binman config get`

### edit
to edit your config run `binman config edit`. This command will make use of whatever editor your $EDITOR var is pointed at.

### add
To add a new repo to your config you can run `binman config add anchore/syft`. This will add `repo: anchore/syft` to our config file in the releases section. If further configuration is required do so with `binman config edit`