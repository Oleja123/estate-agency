package docs

import (
	"bytes"
	"encoding/json"
	"text/template"
)

// ReadDoc renders and returns the swagger document as JSON string.
// This helper is kept in a separate file so it won't be lost when the
// generated docs.go is overwritten by `swag init`.
func ReadDoc() string {
	funcs := template.FuncMap{
		"marshal": func(v interface{}) string {
			b, _ := json.Marshal(v)
			return string(b)
		},
		"escape": func(s string) string {
			b, _ := json.Marshal(s)
			if len(b) >= 2 {
				return string(b[1 : len(b)-1])
			}
			return s
		},
	}
	tmpl, err := template.New("swagger").Funcs(funcs).Parse(docTemplate)
	if err != nil {
		return docTemplate
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, SwaggerInfo); err != nil {
		return docTemplate
	}
	return buf.String()
}
