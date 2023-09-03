package parse

import (
	"log"
	"regexp"
	"strings"

	"github.com/Aize-Public/forego/ctx"
)

type Grammar struct {
	alts map[string]Alt
}

func (this *Grammar) MustAdd(name string, directives string, action any) {
	err := this.Add(name, directives, action)
	if err != nil {
		panic(err)
	}
}

// Add a new production with the given name, directive and action.
// action must be a function with signatures compatible with the directive and return type same as the other production with the same name
func (this *Grammar) Add(name string, directive string, action any) error {
	if this.alts == nil {
		this.alts = map[string]Alt{}
	}
	p := &Prod{
		G:         this,
		Name:      name,
		Directive: directive,
	}
	p.Return(action)
	list := append(this.alts[name], p)
	if list[0].retType != p.retType {
		return ctx.NewErrorf(nil, "first production returns %v, while this one %v", list[0].retType, p.retType)
	}
	this.alts[name] = list
	return nil
}

// build the grammar, returns an error if
func (this *Grammar) Build() error {
	for name, alt := range this.alts {
		log.Printf("build[%q]", name)
		for _, p := range alt {
			err := p.build(this)
			if err != nil {
				return ctx.NewErrorf(nil, "%s: %s: %v", name, p.Directive, err)
			}
		}
	}
	return nil
}

func (this *Grammar) Parse(name string, src []byte) (any, error) {
	p := Pos{
		G:   this,
		src: src,
	}
	alt := this.alts[name]
	out, err := p.ConsumeAlt([]any{}, alt)
	if err != nil {
		return nil, err
	}
	if p.Rem(10) != "" {
		return out, ctx.NewErrorf(nil, "unparsed: %q", p.Rem(80))
	}
	return out, nil
}

func (this *Prod) build(g *Grammar) error {
	//log.Printf("prod[%q]...", this.Name)
	this.Directive = strings.TrimSpace(this.Directive)
	if this.Directive == "" {
		this.act = func(p *Pos) ([]any, error) {
			return nil, nil
		}
		return nil
	}

	var out []func(arg []any, p *Pos) (any, error)
	d := this.Directive
	for len(d) > 0 {
		//log.Printf("parsing %q", d)
		switch d[0] {

		case '[': // TODO
			re := regexp.MustCompile(`\[(.*)\]`)
			m := re.FindString(d)
			panic(m[1])

		case '/':
			re := regexp.MustCompile(`/(([^/]|/.)*)/`)
			m := re.FindStringSubmatch(d)
			if m[0] == "" {
				return ctx.NewErrorf(nil, "invalid directive %q", d)
			}
			d = d[len(m[0]):]
			re, err := regexp.Compile(m[1])
			if err != nil {
				return ctx.NewErrorf(nil, "invalid directive %q: %v", m[1], err)
			}
			//log.Printf("prod[%q]: /%s/", this.Name, re)
			out = append(out, func(arg []any, p *Pos) (any, error) {
				return p.ConsumeRE(re)
			})

		case ' ', '\t', '\n', '\r': // ignore whitespace
			d = d[1:]

		default: // by default, we assume it's the production name
			re := regexp.MustCompile(`(\w+)`)
			m := re.FindStringSubmatch(d)
			if m == nil {
				return ctx.NewErrorf(nil, "invalid directive: %q", d)
			}
			d = d[len(m[0]):]
			name := m[1]
			alt := g.alts[name]
			if alt != nil {
				out = append(out, func(arg []any, p *Pos) (any, error) {
					return p.ConsumeAlt(arg, alt)
				})
			} else {
				return ctx.NewErrorf(nil, "unresolved directive: %q", name)
			}
		}
	}

	this.act = func(p *Pos) ([]any, error) {
		list := []any{}
		for _, f := range out {
			out, err := f(list, p)
			if err != nil {
				return nil, err
			}
			list = append(list, out)
		}
		return list, nil
	}
	return nil
}
