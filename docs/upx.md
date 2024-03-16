### Upx Config

Binman allows for shrinking of your downloaded binaries via [upx](https://upx.github.io/). Ensure upx is in your path and add the following to your binman config to enable shrinking via UPX. Use caution when applying upx to binaries, it can cause unintended side effects.

```yaml
config:
  upx: #Compress binaries with upx
    enabled: false
    args: [] # arrary of args for upx https://linux.die.net/man/1/upx
```