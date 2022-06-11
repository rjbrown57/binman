FROM ghcr.io/rjbrown57/binman:latest AS binman
RUN binman -r "sigstore/cosign"
FROM ubuntu:latest
COPY --from=binman /cosign-linux-amd64 /usr/bin/cosign
RUN chmod 755 /usr/bin/cosign
