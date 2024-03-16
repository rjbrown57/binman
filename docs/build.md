# Binman build subcommand
Binman can build objects from your synced releases. Currently this is limited to OCI Images.

## oci
Images are built by appending synced binaries onto a base image. Each binary will become it's own layer in the final image.

| Flag | Description | Default |
| ----------- | ----------- | ---------- |
| --imageBinPath | Where binaries should be located within the image | /usr/local/bin/ |
| --publishPath| target to publish OCI image to. Should be a valid docker image name. If version is left empty it will be generated | - |
| --repo | Specifc synced repo to add to publish an image for | All releases toolbox |
| --base | Base image to append binaries to | alpine:latest |