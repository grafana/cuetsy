package cuetsy

import (
	"strings"

	"cuelang.org/go/cue"
	tsast "github.com/grafana/cuetsy/ts/ast"
)

func newRootBuilder(ctx *buildContext) *builder {
	return &builder{ctx: ctx}
}

func doit(conf NewConfig, inst *cue.Instance) (result []tsast.Decl, err error) {
	c := &buildContext{
		rootinst:     inst,
		subpath:      conf.Subpath,
		other:        inst,
		externalRefs: map[string]*externalType{},
		schemas:      outputs{},
	}

	iv := inst.Value()
	if c.rootedAtSubpath() {
		iv = iv.LookupPath(c.subpath)
	}

	iter, err := iv.Fields(cue.Optional(true), cue.Definitions(true))
	if err != nil {
		return nil, err
	}

	for iter.Next() {
		label := iter.Selector().String()
		ref := c.makeRef(inst, []string{label})
		if ref == "" {
			continue
		}
		c.schemas.Set(ref, c.build(label, iter.Value()))
	}

	panic("TODO")
}

func (c *buildContext) build(name string, v cue.Value) *tsoutput {
	// TODO should we let errors escape here? Maybe only unsupported-type ones?
	return newRootBuilder(c).enterGen(nil, name, v)
}

func (c *buildContext) makeRef(inst *cue.Instance, ref []string) string {
	ref = append([]string{}, ref...)

	// NOTE this is where oapi does things with its NameFunc

	return strings.Join(ref, ".")
}

func (b *buildContext) rootedAtSubpath() bool {
	return len(b.subpath.Selectors()) > 0
}

func (o outputs) Set(ref string, x *tsoutput) {
	panic("TODO")
}

func (b *builder) enterGen(core *builder, name string, v cue.Value) *tsoutput {
	oldPath := b.ctx.path
	b.ctx.path = append(b.ctx.path, name)
	defer func() { b.ctx.path = oldPath }()

	// NOTE this buncha stuff here related to structural schema - can we skip?
	// var c *builder
	// if core == nil && b.ctx.structural {
	// 	c = newCoreBuilder(b.ctx)
	// 	c.buildCore(v) // initialize core structure
	// 	c.coreSchema()
	// } else {
	// 	c = newRootBuilder(b.ctx)
	// 	c.core = core
	// }

	return b.fill(v)
}

func (b *builder) fill(v cue.Value) *tsoutput {
	if b.filled != nil {
		return b.filled
	}

	// NOTE this sets the string text to be used in the oapi "type" field
	// b.setValueType(v)
	b.setTargetKind(v)

	// isRef := b.value(v, nil)
	panic("TODO")
}

type typeFunc func(b *builder, a cue.Value)

func (b *builder) value(v cue.Value, f typeFunc) (isRef bool) {
	// NOTE oapi does cycle detection bookkeeping here :sad:
	panic("TODO")
}

func (b *builder) setTargetKind(v cue.Value) {
	// TODO decide whether we mix kind information with reference information
	b.tsk, _ = getKindFor(v)
}
