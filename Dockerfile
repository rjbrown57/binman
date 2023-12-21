FROM alpine:latest 
RUN apk add ca-certificates 
COPY binman /usr/local/bin/binman
