package binman

import (
	"bytes"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
)

// Format strings for processing. Currently used by releaseFileName and DlUrl
func formatString(templateString string, dataString string) string {

	// For compatability with previous binman versions update %s to {{.}}
	templateString = strings.Replace(templateString, "%s", "{{.}}", -1)

	// we need an io.Writer to capture the template output
	buf := new(bytes.Buffer)

	// https://github.com/Masterminds/sprig use spring functions for extra templating functions
	tmpl, err := template.New("stringFormatter").Funcs(sprig.FuncMap()).Parse(templateString)
	if err != nil {
		log.Fatalf("unable to process template for %s", templateString)
	}

	err = tmpl.Execute(buf, dataString)
	if err != nil {
		log.Fatalf("unable to process template for %s", templateString)
	}

	return buf.String()
}
