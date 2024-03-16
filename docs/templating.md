# String Templating

Binman supports templating via go templates and [sprig](https://masterminds.github.io/sprig/).

Templating is available on the following fields

* url
* releasefilename
* postcommands args

The following values are provided
| key | notes |
| ----------- | ----------- |
| os | the configured os. Usually the os of your workstation |
| arch | the configured architecture. Usually the arch of your workstation
| version | the asset version we have fetched from github |
| project | the github project name |
| org | the github org name |
| artifactpath | the full path to the final extracted release artifact. * |
| link | the full path to link binman creates. * |
| filename | just the file name of the final release artifact. * |

\* these values are only available to args in postcommands actions.