package encoder

import "text/template"

// Generate a typescript enum declaration. Inputs:
// .maturity	Optional. Maturity level, applied in doc comment
// .export		Whether to export the declaration
// .name		The name of the typescript enum
// .pairs		Slice of {K: string, V: string}
var enumCode = template.Must(template.New("enum").Parse(`
{{if .maturity -}}
/**
 * {{.maturity}}
 */
{{- end}}
{{if .export}}export{{end}} enum {{.name}} {
  {{- range .pairs}}
  {{.K}} = {{.V}},{{end}}
}
`))

// Generate a typescript interface declaration. Inputs:
// .maturity	Optional. Maturity level, applied in doc comment
// .name		The name of the typescript enum.
// .pairs		Slice of {K: string, V: string}
var interfaceCode = template.Must(template.New("enum").Parse(`
{{if .maturity -}}
/**
 * {{.maturity}}
 */
{{end -}}
{{if .export}}export{{end}} interface {{.name}} {
  {{range .pairs}}{{.K}}: {{.V}},{{end}}
}
`))
