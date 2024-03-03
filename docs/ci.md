# CI use cases

Binman has a few use cases in CI piplines.

## Using "ghcr.io/rjbrown57/binman:latest" container image

You can use the binman image in a multi-stage build to grab binaries for docker images.

```Dockerfile
FROM ghcr.io/rjbrown57/binman:latest AS binman
RUN binman get "sigstore/cosign"
FROM ubuntu:latest
COPY --from=binman /cosign-linux-amd64 /usr/bin/cosign
RUN chmod 755 /usr/bin/cosign
```

Or use with a config file to grab multiple binaries
```Dockerfile
FROM ghcr.io/rjbrown57/binman:latest AS binman
ADD myconfig.yaml examples/basicExample.yaml
RUN binman -c examples/basicExample.yaml
FROM ubuntu:latest
COPY --from=binman /root/binMan/ /usr/local/bin/
```

## Building Toolbox Images

The binman build command can be used to create minimal images that are often useful to run ci pipelines. See [build subcommand](../docs/build.md)