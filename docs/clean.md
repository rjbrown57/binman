# Binman clean subcommand
Binman can remove old releases with the binman clean subcommand. The following options are supported.

| Flag | Description | Default |
| ----------- | ----------- | ---------- |
| -r,--dryrun | enable dry run to check what will be removed | false |
| -s,--scan | update db with local files pre scan. Useful if binman has been used previous to 1.0.0 | false |
| -n,--threshold |  Non-zero amount of releases to retain | 3 |