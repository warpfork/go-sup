package sup_test

import (
	"bytes"
	"testing"
	"text/template"
)

func mustEqual(t *testing.T, actual, expect interface{}) {
	t.Helper()
	if actual != expect {
		t.Fatalf("%+v != %+v", actual, expect)
	}
}

func shouldEqual(t *testing.T, actual, expect interface{}) {
	t.Helper()
	if actual != expect {
		t.Errorf("%+v != %+v", actual, expect)
	}
}

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
