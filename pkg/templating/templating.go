package templating

import (
	"bytes"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	log "github.com/rjbrown57/binman/pkg/logging"
)

// Format strings for processing. Currently used by releaseFileName and DlUrl
func TemplateString(templateString string, dataMap map[string]interface{}) string {

	// For compatability with previous binman versions update %s to {{.}}
	templateString = strings.Replace(templateString, "%s", "{{.version}}", -1)

	// we need an io.Writer to capture the template output
	buf := new(bytes.Buffer)

	// https://github.com/Masterminds/sprig use sprig functions for extra templating functions
	tmpl, err := template.New("stringFormatter").Funcs(sprig.FuncMap()).Parse(templateString)
	if err != nil {
		log.Fatalf("unable to process template for %s", templateString)
	}

	err = tmpl.Execute(buf, dataMap)
	if err != nil {
		log.Fatalf("unable to process template for %s", templateString)
	}

	return buf.String()
}
