package parse

import (
	"fmt"
	"regexp"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/Aize-Public/forego/ctx"
)

type Grammar struct {
	m sync.Mutex

	// Trailing regexp
	End *regexp.Regexp

	built atomic.Bool
	alts  map[string]Alt
	Log   func(f string, args ...any)
}

var Whitespaces = regexp.MustCompile(`[\s\n\r]*`)
var CommentsAndWhitespaces = regexp.MustCompile(`(\s|//[^\n]*\n?)*`)

// Add a new production with the given name, directive and action.
// action must be a function with signatures compatible with the directive and return type same as the other production with the same name
// panics if anything is wrong (you normally don't want to handle the error, since can be seen as a compile time error)
func (this *Grammar) Add(name string, directives string, action any) *Prod {
	if this.Log != nil {
		this.Log("adding %s: %s", name, directives)
	}
	this.built.Store(false)
	if this.alts == nil {
		this.alts = map[string]Alt{}
	}
	_, file, line, _ := runtime.Caller(1)
	p := &Prod{
		g:         this,
		Name:      name,
		Directive: directives,
		src:       fmt.Sprintf("%s:%d", file, line),
	}
	p.Return(action)
	list := append(this.alts[name], p)
	this.alts[name] = list
	return p
}

// build the grammar, returns an error if the grammar is not complete
func (this *Grammar) Build() error {
	this.m.Lock()
	defer this.m.Unlock()
	if this.Log != nil {
		this.Log("building...")
	}
	this.built.Store(true)
	for name, alt := range this.alts {
		if this.Log != nil {
			this.Log("build[%q]", name)
		}
		for _, p := range alt {
			err := p.build(this)
			if err != nil {
				return ctx.NewErrorf(nil, "%s: %s: %v", name, p.Directive, err)
			}
		}
	}
	return nil
}

// parse the given text, optionally compile the grammar if needed
func (this *Grammar) Parse(name string, text []byte) (any, error) {
	if !this.built.Load() { // already built
		err := this.Build()
		if err != nil {
			return nil, err
		}
	}
	p := Pos{
		g:   this,
		src: text,
	}
	out, err := p.ConsumeAlt(this.alts[name])
	if err != nil {
		return nil, err
	}
	if this.End != nil {
		p.IgnoreRE(this.End)
	}
	if p.Rem(10) != "" {
		return out, ctx.NewErrorf(nil, "unparsed: %q", p.Rem(80))
	}
	return out, nil
}
