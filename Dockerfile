FROM ubuntu:latest 
RUN apt-get update && \
apt-get -yqq install ca-certificates && \
rm -rf /var/cache/apt/lists
COPY binman /usr/local/bin/binman
