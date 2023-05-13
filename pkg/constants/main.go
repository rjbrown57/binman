package constants

// Common regexes
const TarRegEx = `(\.tar$|\.tar\.gz$|\.tgz$)`
const ZipRegEx = `(\.zip$)`
const ExeRegex = `.*\.exe$`
const X86RegEx = `(amd64|x86_64)`
const MacOsRx = `(darwin|macos|apple)`

// Url defaults
// must have /
const DefaultGHBaseURL = "https://api.github.com/"

// no /
const DefaultGLBaseURL = "https://gitlab.com"
