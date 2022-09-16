package ast

// Export wraps the provided Decl with an export keyword.
// Comments from above the provided Decl are hoisted.
func Export(decl Decl) Decl {
	e := ExportKeyword{Decl: decl}
	if comm, is := decl.(Commenter); is {
		e.Comment = comm.hoistComments()
	}
	return e
}
