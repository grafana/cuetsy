package encoder

import (
	"strings"
	"text/template"
)

// Generate a typescript type declaration. Inputs:
// .maturity	Optional. Maturity level, applied in doc comment
// .export		Whether to export the declaration
// .name		  The name of the typescript type
// .tokens		Slice of token strings
var typeCode = template.Must(template.New("enum").
	Funcs(template.FuncMap{"Join": strings.Join}).Parse(`
{{if .maturity -}}
/**
 * {{.maturity}}
 */
{{end -}}
{{if .export}}export {{end -}}
type {{.name}} = {{ Join .tokens " | "}}
`))

// Generate a typescript enum declaration. Inputs:
// .maturity	Optional. Maturity level, applied in doc comment
// .export		Whether to export the declaration
// .name		  The name of the typescript enum
// .pairs		  Slice of {K: string, V: string}
var enumCode = template.Must(template.New("enum").Parse(`
{{if .maturity -}}
/**
 * {{.maturity}}
 */
{{end -}}
{{if .export}}export {{end -}}
enum {{.name}} {
  {{- range .pairs}}
  {{.K}} = {{.V}},{{end}}
}
`))

// Generate a typescript interface declaration. Inputs:
// .maturity	Optional. Maturity level, applied in doc comment
// .name		The name of the typescript enum.
// .pairs		Slice of {K: string, V: string}
// .extends		Slice of other interface names to extend
var interfaceCode = template.Must(template.New("interface").
	Funcs(template.FuncMap{"Join": strings.Join}).Parse(`
{{if .maturity -}}
/**
 * {{.maturity}}
 */
{{end -}}
{{if .export}}export {{end -}}
interface {{.name}}{{if ne (len .extends) 0}} extends {{ Join .extends ", "}}{{end}} {
  {{- range .pairs}}
  {{.K}}: {{.V}},{{end}}
}
`))
