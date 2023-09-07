package parse

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/Aize-Public/forego/ctx"
)

type Alt []*Prod

type Prod struct {
	g *Grammar

	// what will be removed before consuming any directive
	WS *regexp.Regexp

	// Set by Add()
	Name string

	// Set by Add()
	Directive string

	// file:line where this production was Add()-ed
	src string

	// each directive part will generate
	actions []action

	// function to be used at the end of the production
	ret func(from int, at *pos, in []any) (any, error)
}

type action struct {
	// if true, results won't be added to the output
	silent bool

	commit bool

	p    *Prod
	prod string
	re   *regexp.Regexp
}

func (this action) String() string {
	s := ""
	if this.commit {
		return "!"
	}
	if this.silent {
		s = "~"
	}
	if this.re != nil {
		return s + "/" + this.re.String() + "/"
	}
	if this.prod != "" {
		return s + `{` + this.prod + `}`
	}
	return s
}

func (this action) exec(p *pos) (any, *Error) {
	if this.commit {
		p.Log("commit %p", p)
		p.commit = true
		return nil, nil
	}
	if this.re != nil {
		out, err := p.ConsumeRE(this.re)
		return out, err
	}
	if this.prod != "" {
		alt := this.p.g.alts[this.prod]
		if len(alt) == 0 {
			return nil, p.NewErrorf("no prod with name %q", this.prod)
		}
		return p.ConsumeAlt(alt)
	}
	return nil, p.NewErrorf("empty action")
}

// helper
func (this *Prod) Parse(in []byte, end *regexp.Regexp) (any, error) {
	p := &pos{
		g:   this.g,
		src: in,
	}
	out, err := p.ConsumeAlt(Alt{this})
	if err != nil {
		return nil, err
	}
	if end != nil {
		p.IgnoreRE(end)
	}
	if p.Rem(10) != "" {
		return out, ctx.NewErrorf(nil, "rem: %q", p.Rem(80))
	}
	return out, nil
}

func (this *Prod) build() error {
	//log.Printf("prod[%q]...", this.Name)
	this.Directive = strings.TrimSpace(this.Directive)
	if this.Directive == "" {
		//this.act = func(p *Pos) ([]any, error) {
		//	return nil, nil
		//}
		return nil
	}

	d := this.Directive
	for len(d) > 0 {
		//log.Printf("parsing %q", d)
		switch d[0] {

		case '!': // commit to this production
			this.actions = append(this.actions, action{
				p:      this,
				commit: true,
				silent: true,
			})
			d = d[1:]

		case '[': // TODO
			re := regexp.MustCompile(`\[(.*)\]`)
			m := re.FindString(d)
			panic(m[1])

		case '"':
			re := regexp.MustCompile(`^"(([^"\\]|\\.)*)"`)
			m := re.FindStringSubmatch(d)
			if m == nil {
				return ctx.NewErrorf(nil, "invalid directive `%s`", d)
			}
			d = d[len(m[0]):]
			re = regexp.MustCompile(regexp.QuoteMeta(m[1]))
			this.actions = append(this.actions, action{
				p:      this,
				re:     re,
				silent: true,
			})

		case '/':
			re := regexp.MustCompile(`^/(([^/\\]|\\.)*)/`)
			m := re.FindStringSubmatch(d)
			if m == nil {
				return ctx.NewErrorf(nil, "invalid directive `%s`", d)
			}
			d = d[len(m[0]):]
			re, err := regexp.Compile(m[1])
			if err != nil {
				return ctx.NewErrorf(nil, "invalid directive `%s`: %v", m[1], err)
			}
			//log.Printf("prod[%q]: /%s/", this.Name, re)
			this.actions = append(this.actions, action{
				p:  this,
				re: re,
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
			this.actions = append(this.actions, action{
				p:    this,
				prod: name,
			})
		}
	}

	return nil
}

func (this *Prod) verify() error {
	for _, act := range this.actions {
		if act.prod != "" {
			if len(this.g.alts[act.prod]) == 0 {
				return ctx.NewErrorf(nil, "production %q `%s` refers to empty %q", this.Name, this.Directive, act.prod)
			}
		}
	}
	return nil
}

func (this *Prod) exec(p *pos) (any, *Error) {
	list := make([]any, 0, len(this.actions))
	var err *Error
	from := p.at
	for _, act := range this.actions {
		if this.WS != nil {
			err := p.IgnoreRE(this.WS)
			if err != nil {
				return nil, p.NewErrorf("can't consume whitespace: %v", err)
			}
		}

		out, err := act.exec(p)
		if err != nil {
			return nil, err
		}
		if !act.silent {
			list = append(list, out)
		}
	}
	if this.ret == nil {
		switch len(list) {
		case 0:
			return nil, err
		case 1:
			return list[0], err
		default:
			return list, err
		}

	} else {
		//if this.G.Log != nil { this.G.Log("ret(%v, %v)", in, out) }
		out, err := this.ret(from, p, list)
		if err != nil {
			return out, &Error{err, p.at}
		}
		p.Log("return %v", out)
		return out, nil
	}
}

// set a new return
func (this *Prod) Return(action any) *Prod {
	if action == nil {
		this.ret = func(from int, p *pos, in []any) (any, error) {
			switch len(in) {
			case 0:
				return nil, nil
			case 1:
				return in[0], nil
			default:
				return in, nil
			}
		}
		return this
	}
	f := reflect.ValueOf(action)
	t := f.Type()

	actNum := 0
	for _, act := range this.actions {
		if !act.silent {
			actNum++
		}
	}
	wantPos := t.NumIn() > 0 && t.In(0) == reflect.TypeOf(Pos{})
	if wantPos {
		actNum++
	}

	if t.NumIn() != actNum {
		panic(fmt.Sprintf("%s: %v expects %d args, but %d are in the directive (%+v)", this.src, t, t.NumIn(), actNum, this.actions))
	}
	switch t.NumOut() {
	case 1:
	case 2:
		if t.Out(1) != reflect.TypeOf((*error)(nil)).Elem() {
			panic(fmt.Sprintf("%s: %v should return (X, error) or (X)", this.src, t))
		}
	default:
		panic(fmt.Sprintf("%s: %v should return (X, error) or (X)", this.src, t))
	}
	// TODO: test out either (X, error) or (X)

	this.ret = func(from int, p *pos, in []any) (any, error) {
		if this.g.Log != nil {
			ins := []string{}
			for _, in := range in {
				ins = append(ins, fmt.Sprintf("%T", in))
			}
			this.g.Log("calling `%v` with (%s)", t, strings.Join(ins, ", "))
		}
		var list []reflect.Value
		if wantPos {
			list = append(list, reflect.ValueOf(Pos{from, p.at, p.src}))
		}
		for _, in := range in {
			t := t.In(len(list)) // expected type
			v, err := coerce(reflect.ValueOf(in), t)
			if err != nil {
				panic(fmt.Sprintf("%s: can't coerce arg #%d: %v", this.src, len(list), err))
			}
			//if this.g.Log != nil { this.g.Log("ARG: %v", v) }
			list = append(list, v)
		}
		if len(list) != t.NumIn() {
			panic(fmt.Sprintf("%s: action `%v` expects %d arguments, but got %+v (wantPos: %v)", this.src, t, t.NumOut(), len(list), wantPos))
		}
		//log.Printf("CALL[%v](%v)", f, list)
		out := f.Call(list)
		switch len(out) {
		case 1:
			return out[0].Interface(), nil
		case 2:
			first := out[0].Interface()
			second := out[1].Interface()
			if second == nil {
				return first, nil
			}
			if this.g.Log != nil {
				this.g.Log("ERR: %T (%+v)", second, second.(error))
			}
			return first, second.(error)
		default:
			panic("can only return (any) or (any, error)")
		}
	}
	return this
}
