package parse

import (
	"fmt"
	"io"
	"reflect"
	"regexp"
	"strings"

	"github.com/Aize-Public/forego/ctx"
)

// lowercase?
type alt []*Prod

func (this alt) weight(maxDepth int) float64 {
	if maxDepth == 0 {
		return 1
	}
	x := 1.0
	for _, p := range this {
		x *= p.weight(maxDepth)
	}
	return x
}

func (this Prod) weight(maxDepth int) float64 {
	w := 1.0
	for _, act := range this.actions {
		if act.commit {
			return w / 2
		}
		if act.prod != "" {
			w = w*2 + act.p.g.alts[act.prod].weight(maxDepth-1)*0.75
		}
	}
	return w
}

type Prod struct {
	g *Grammar

	// what to ignore before any text matching
	WS *regexp.Regexp

	// override the above
	wsFrom *Prod

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

	p        *Prod
	prod     string
	re       *regexp.Regexp
	negative bool // if true, make into a negative lookahead
}

func (this action) String() string {
	s := ""
	if this.commit {
		return "+"
	}
	if this.silent {
		s = "~"
	}
	if this.re != nil {
		return s + "/" + this.re.String() + "/"
	}
	if this.prod != "" {
		return s + this.prod
	}
	return s
}

func (this *Prod) ws() *regexp.Regexp {
	if this.wsFrom != nil {
		return this.wsFrom.ws()
	}
	return this.WS
}

func (this action) exec(p *pos) (any, *Error) {
	if this.commit {
		p.Log("commit %p", p)
		p.commit = true
		return nil, nil
	}
	if this.re != nil {
		ws := this.p.ws()
		if ws != nil {
			err := p.IgnoreRE(ws, false)
			if err != nil {
				return nil, p.NewErrorf("can't consume whitespace: %v", err)
			}
		}

		out, err := p.ConsumeRE(this.re, this.negative)
		return out, err
	}
	if this.prod != "" {
		alt := this.p.g.alts[this.prod]
		if len(alt) == 0 {
			return nil, p.NewErrorf("no prod with name %q", this.prod)
		}
		return p.consumeAlt(alt)
	}
	return nil, p.NewErrorf("empty action")
}

// helper
func (this *Prod) Parse(fname string, in []byte, end *regexp.Regexp) (any, error) {
	p := &pos{
		g:     this.g,
		file:  fname,
		src:   &Src{bytes: in},
		stats: &Stats{},
	}
	out, err := p.consumeAlt(alt{this})
	if err != nil {
		return nil, err
	}
	if end != nil {
		p.IgnoreRE(end, false)
	}
	if p.Rem(10) != "" {
		return out, ctx.NewErrorf(nil, "rem: %q", p.Rem(80))
	}
	return out, nil
}

func parseText(d string) (*regexp.Regexp, int, error) {
	re := regexp.MustCompile(`^"(([^"\\]|\\.)*)"`)
	m := re.FindStringSubmatch(d)
	if m == nil {
		return nil, 0, ctx.NewErrorf(nil, "invalid directive `%s`", d)
	}
	re = regexp.MustCompile("^" + regexp.QuoteMeta(m[1]))
	return re, len(m[0]), nil
}

func parseRE(d string) (*regexp.Regexp, int, error) {
	reEnd := regexp.MustCompile(`^/(([^/\\]|\\.)*)/`)
	m := reEnd.FindStringSubmatch(d)
	if m == nil {
		return nil, 0, ctx.NewErrorf(nil, "invalid directive `%s`", d)
	}
	d = d[len(m[0]):]
	re, err := regexp.Compile("^" + m[1])
	if err != nil {
		return nil, 0, ctx.NewErrorf(nil, "invalid directive `%s`: %v", m[1], err)
	}
	return re, len(m[0]), nil
}

func (this *Prod) mustBuild(term string) int {
	ct, err := this.build(term)
	if err != nil {
		panic(err)
	}
	return ct
}

func (this *Prod) build(term string) (int, error) {
	//log.Printf("prod[%q]...", this.Name)
	this.Directive = strings.TrimSpace(this.Directive)
	if this.Directive == "" {
		//this.act = func(p *Pos) ([]any, error) {
		//	return nil, nil
		//}
		return 0, nil
	}

	negative := false
	silent := false
	d := this.Directive
	for {
		//log.Debugf(nil, "REM: `%s`", d)
		switch term {
		case "":
			if d == "" {
				return 0, nil
			}
		default:
			if strings.HasPrefix(d, term) {
				return len(this.Directive) - len(d), nil
			}
		}
		if len(d) == 0 {
			return 0, io.EOF
		}
		//log.Printf("parsing %q", d)
		switch d[0] {

		case '~':
			silent = true
			d = d[1:]

		case '!': // negative look ahead
			negative = true
			d = d[1:]

		case '+': // commit to this production
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
			re, ct, err := parseText(d)
			if err != nil {
				return len(this.Directive) - len(d), nil
			}
			d = d[ct:]
			this.actions = append(this.actions, action{
				p:        this,
				re:       re,
				negative: negative,
				silent:   true,
			})
			negative = false
			silent = false

		case '/':
			re, l, err := parseRE(d)
			if err != nil {
				return len(this.Directive) - len(d), nil
			}
			d = d[l:]
			//log.Printf("prod[%q]: /%s/", this.Name, re)
			this.actions = append(this.actions, action{
				p:        this,
				re:       re,
				negative: negative,
				silent:   silent,
			})
			negative = false
			silent = false

		case ' ', '\t', '\n', '\r': // ignore whitespace
			d = d[1:]

		default: // by default, we assume it's the production name
			re := regexp.MustCompile(`(\w+)`)
			m := re.FindStringSubmatch(d)
			if m == nil {
				return len(this.Directive) - len(d), ctx.NewErrorf(nil, "invalid directive: %q", d)
			}
			d = d[len(m[0]):]
			name := m[1]

			if len(d) > 2 {
				//log.Debugf(nil, "REPEAT: `%s`", d)
				switch d[0:1] {
				case "(":
					d = d[1:]
					if negative {
						return 0, ctx.NewErrorf(nil, "can't do a negative lookahead with repetition")
					}
					temp := &Prod{
						Directive: d,
					}
					ct, err := temp.build(")")
					if err != nil {
						return len(this.Directive) - len(d), ctx.NewErrorf(nil, "invalid repetition: %v", err)
					}

					// internal names for the new alternations
					repName := fmt.Sprintf("%s,rep%d", this.Name, this.g.repCt.Add(1))
					repRep := repName + "_"

					rep := d[0:ct]
					d = d[ct+1:]
					//log.Debugf(nil, "REPEAT: %+v", temp)
					var sepAction *action
					switch len(temp.actions) {
					case 1: // simple
					case 2: // with separator
						sepAction = &temp.actions[1]
					default:
						return 0, ctx.NewErrorf(nil, "invalid repetition: `%s`", rep)
					}

					// TODO(oha) allow for other options like `?` or `s?` or `3` or `3..5` or `..5` etc
					switch temp.actions[0].prod {
					case "s":
					default:
						return 0, ctx.NewErrorf(nil, "expected valid repetition, got `%s`", rep)
					}

					// external prod, catches `name` and then repetitions
					p1 := &Prod{
						g:         this.g,
						Directive: name + " " + repRep,
						Name:      repName,
						src:       this.src,
						wsFrom:    this,
					}
					p1.actions = append(p1.actions, action{p: p1, prod: name})
					p1.actions = append(p1.actions, action{p: p1, prod: repRep})
					p1.Return(func(l any, r []any) []any {
						return append([]any{l}, r...)
					})

					// internal prod, catches `sep` and `name` and then itself
					p2 := &Prod{
						g:      this.g,
						Name:   repRep,
						src:    this.src,
						wsFrom: this,
					}
					p2.Directive = name + " " + repRep
					if sepAction != nil {
						sepAction.p = p2
						p2.actions = append(p2.actions, *sepAction)
						p2.Directive = sepAction.prod + " " + p2.Directive
					}
					p2.actions = append(p2.actions, action{p: p2, prod: name})
					p2.actions = append(p2.actions, action{p: p2, prod: repRep})

					if sepAction != nil && !sepAction.silent {
						p2.Return(func(sep any, l any, r []any) []any {
							return append([]any{sep, l}, r...)
						})
					} else {
						p2.Return(func(l any, r []any) []any {
							return append([]any{l}, r...)
						})
					}

					// empty fallback, when reaching the end
					p3 := &Prod{
						g:         this.g,
						Directive: "",
						Name:      repRep,
						src:       this.src,
						wsFrom:    this,
					}
					p3.mustBuild("")
					p3.Return(func() []any { return []any{} })
					this.g.alts[repName] = alt{p1}
					this.g.alts[repRep] = alt{p2, p3}

					name = repName // replace name with repName to use the above

				default:
				}

			}
			this.actions = append(this.actions, action{
				p:        this,
				prod:     name,
				negative: negative,
				silent:   silent,
			})
			negative = false
			silent = false
		}
	}
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
	p.p = this
	list := make([]any, 0, len(this.actions))
	var err *Error
	from := p.at
	for _, act := range this.actions {
		//if act.negative {
		//	rev := p.at
		//	_, err := act.exec(p)
		//	p.at = rev
		//	if err == nil {
		//		return nil, p.NewErrorf("negative lookahead: %s", p.Rem(20))
		//	}
		//	p.Log("negative lookahead ok: %v", err)
		//} else {
		out, err := act.exec(p)
		if err != nil {
			if err.commit {
				// committed error must return directly
				return nil, err
			}
			if p.commit {
				// if we are committed, but the error isn't, wrap it so it's easier to see where the commit happened
				err = p.NewErrorf("expected %s got %q", act.String(), p.Rem(10))
				err.commit = true // and commit!
			}
			// return a non-committed error
			return nil, err
		}
		if !act.silent {
			list = append(list, out)
		}
		//}
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
			return out, &Error{err, p.at, p.commit}
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
			p.Log("calling `%v` with (%s)", t, strings.Join(ins, ", "))
		}
		var list []reflect.Value
		if wantPos {
			list = append(list, reflect.ValueOf(Pos{from, p.at, p.file, p.src}))
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
