![intro](logo/binmanlogo.png)

## General

Binman is a tool to sync release assets from github or gitlab to your local workstation. The main use case is syncing via config to keep the tools you use every day up to date.

![demo](examples/demo.gif)

Grab the latest release [here](https://github.com/rjbrown57/binman/releases), and let binman grab it for you next time :rocket:

Binman will attempt to find a release asset that matches your OS and architecture and one of the types of files we handle currently. Currently handled file types are "zip", "tar", "binary", "exe".

Just add the releasepath to your shell PATH var and you are good to go!

## Config Sync

To run binman effectively you need a config. 

Running binman with no arguments, and no config will populate the default config to your OS's appropriate [config directory](https://pkg.go.dev/os#UserConfigDir). On linux the config file will be added to `~/.config/binman/config`.

Binman also allows supplying a configfile from an alternate path with the `-c` flag or by the "BINMAN_CONFIG" environment variable.

Here's an example config file

```yaml
config:
  releasepath:  #path to keep fetched releases. $HOME/binMan is the default
  cleanup: true # remove downloaded archive
  maxdownloads: 1 # number of concurrent downloads allowed. Default is 3
  sources:
   - name: gitlab.com
     #tokenvar: GL_TOKEN # environment variable that contains gitlab token
     apitype: gitlab
   - name: github.com
     #tokenvar: GH_TOKEN # environment variable that contains github token
     apitype: github
releases:
  - repo: rjbrown57/binman
  # syncing from gitlab
  - repo: gitlab.com/gitlab-org/cli
```

If you find a new binary you would like to add to your config file add it via your editor of choice with `binman config edit`, or take a shortcut with `binman config add therepo/project`.

Binman can also run with a "contextual" config file. If a directory contains a file ".binMan.yaml" this will be merged with your main config. Check a config into git projects and easily fetch required dependencies.

Binman has many config options. To read more about those check out the [config docs](docs/config.md)

## Docs

| Doc | Description |
|-----|------|
| [Config Options](docs/config.md) | Details on the many config options for binman |
| [Server SubCommand](docs/server.md) | Running in server mode. This allows you to point your binman client at an internal server and avoid gh/gl limits or external traffic |
| [Clean Subcommand](docs/clean.md) | The clean subcommand is used to remove old releases |
| [Build Subcommand](docs/build.md) | The build subcommand can be used to create OCI images of synced releases quickly |
| [CI Usage](docs/ci.md)| Docs on potential use-cases for binman in CI|