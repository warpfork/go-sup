package sup_test

import (
	"bytes"
	"text/template"
)

func tmplToStr(tmpl string, obj interface{}) string {
	var buf bytes.Buffer
	if err := template.Must(
		template.New("").
			Parse(tmpl),
	).Execute(&buf, obj); err != nil {
		panic(err)
	}
	return buf.String()
}

func mapToStr(foo interface{}) string {
	// easier to implement with template, since that also handles sorting.
	return tmplToStr(
		`{{range $k, $v := . -}}`+
			`{{printf "  - %q: %v\n" $k $v}}`+
			`{{end}}`,
		foo,
	)
}
