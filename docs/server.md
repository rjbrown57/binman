# Binman Server

If you want to run binman continuously, configure the watch portion of your config file and invoke `binman watch`. This will also expose `/metrics` and `/healthz` and the query api on `/v1/` on port 9091 by default. An example config is below.

```yaml
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
      fileserver: true # Start binman in server mode
      frequency: 60 # seconds between iteration, default is 60
      port: 9091 # Port to expose healthz and metrics endpoints, default is 9091
releases:
- repo: rjbrown57/binman
```

metrics are exposed in the format `binman_release{latest="true",repo="rjbrown57/binman",version="v0.8.0"} 0`. Keep in mind github api limits when configuring how often binman checks for new assets.


## Pointing binman clients at your binman server

Running binman as a server will allow you to run a centralized binman server, and avoid reaching out to github/gitlab directly. See the helm chart for suggestions on how to run in kubernetes.

With a running server you can populate your config like the below to sync repos from the binman server instance.

```yaml
defaults:
  source: "binmank8s" # If this default is set all releases will search your target binman
config:
  sources:
   - name: binmank8s
     apitype: binman
     url: http://binman.default.svc.cluster.local:9091
releases:
  - repo: rjbrown57/binman
  - repo: anchore/syft
  - repo: gitlab.com/gitlab-org/cli
```

## Running in k8s

An example helm chart is provided [here for running in k8s](../charts/binman/)

## Running with docker-commpose

Binman can also easily be run with docker-compose. See [docker-compose](../examples/docker-compose.yaml)