package encoder

type builder struct {
}

// func (b *builder) setValueType(v cue.Value) {
// 	if b.core != nil {
// 		return
// 	}

// 	switch v.IncompleteKind() {
// 	case cue.BoolKind:
// 		b.typ = "boolean"
// 	case cue.FloatKind, cue.NumberKind:
// 		b.typ = "number"
// 	case cue.IntKind:
// 		b.typ = "integer"
// 	case cue.BytesKind:
// 		b.typ = "string"
// 	case cue.StringKind:
// 		b.typ = "string"
// 	case cue.StructKind:
// 		b.typ = "object"
// 	case cue.ListKind:
// 		b.typ = "array"
// 	}
// }
