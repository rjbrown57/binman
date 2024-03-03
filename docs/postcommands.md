# Post Commands

binman supports executing arbitrary os commands after it has fetched a release artifact. The templating detailed in [string templating](../docs/templating.md) is available to postcommand args. A simple example is to copy the file to a new location.

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