package cuetsy

import (
	"cuelang.org/go/cue"
	"cuelang.org/go/cue/errors"
	tsast "github.com/grafana/cuetsy/ts/ast"
)

type NewConfig struct {
	// Subpath is the path within the cue.Instance that cuetsy should treat as the
	// root object to be rendered. Optional.
	Subpath cue.Path

	// Self indicates that, rather than rendering all the attribute-annotated
	// children of provided instance, the instance itself should be rendered.
	Self bool

	// InjectCuetsyAttrs is an optional set of attribute bodies to be dynamically
	// injected onto values as they're being rendered.
	//
	// The keys will be passed to cue.ParsePath(). The value will be treated as
	// the body of a @cuetsy attribute (@cuetsy(<value>)) for values encountered at
	// that path, relative to the root.
	InjectCuetsyAttrs map[string]string
}

type buildContext struct {
	// The root Value being converted into TS
	// root cue.Value // TODO needs to be a cue.Instance?
	rootinst *cue.Instance // TODO needs to be a cue.Instance?
	// Path to root value to render within root instance
	subpath cue.Path
	// A value not within the same as the root value that's being processed
	other *cue.Instance
	// Path currently being processed
	path []string
	// Errors accumulated along the way during build
	errs errors.Error

	// List of external types, not within the tree of the root value
	externalRefs map[string]*externalType

	schemas outputs

	// ??
	// refPrefix string // appears to just be for "components/schema" path prefix in oapi
}

type outputs map[string]*tsoutput

type tsoutput struct {
	typ    tsast.Decl
	defaul tsast.Decl
}

type builder struct {
	ctx    *buildContext
	typ    string
	format string

	// right type??
	singleFields tsast.Decl
	current      []tsast.Decl

	tsk  tsKind
	kind cue.Kind

	filled *tsoutput
	values []cue.Value
	keys   []string

	// i think ?? these are all in reference to the subtrees openapi demands
	core       *builder
	properties map[string]*builder
	items      *builder
}

type externalType struct {
	ref   string
	inst  *cue.Instance
	path  []string
	value cue.Value
}
