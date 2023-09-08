package parse

import (
	"fmt"
	"regexp"
	"runtime"
	"time"

	"github.com/Aize-Public/forego/ctx"
)

type Grammar struct {

	// Trailing regexp
	End *regexp.Regexp

	alts map[string]Alt
	Log  func(f string, args ...any)

	Stats struct {
		Productions  int
		Alternations int
		ParseCt      int
		ParseElapsed time.Duration
	}
}

func (this Grammar) String() string {
	return fmt.Sprintf("parse.Grammar{%d/%d %v}", this.Stats.Productions, this.Stats.Alternations, this.Stats.ParseElapsed/time.Duration(this.Stats.ParseCt))
}

var Whitespaces = regexp.MustCompile(`[\s\n\r]*`)
var CommentsAndWhitespaces = regexp.MustCompile(`(\s|//[^\n]*\n?)*`)

// Add a new production with the given name and directive
// if many elements are in the directive, it returns a list of objects
// otherwise return the only element
// returns a production that can further be tweaked, adding a Return() action which override the above, and changing the whitespace
// panics if anything is wrong (you normally don't want to handle the error, since can be seen as a compile time error)
func (this *Grammar) Add(name string, directives string) *Prod {
	if this.Log != nil {
		this.Log("adding %s: %s", name, directives)
	}
	if this.alts == nil {
		this.Stats.Alternations++
		this.alts = map[string]Alt{}
	}
	this.Stats.Productions++
	_, file, line, _ := runtime.Caller(1)
	p := &Prod{
		g:         this,
		Name:      name,
		Directive: directives,
		src:       fmt.Sprintf("%s:%d", file, line),
	}
	err := p.build()
	if err != nil {
		panic(err)
	}
	list := append(this.alts[name], p)
	this.alts[name] = list
	return p
}

// build the grammar, returns an error if the grammar is not complete
func (this *Grammar) Verify() error {
	for name, alt := range this.alts {
		if this.Log != nil {
			this.Log("build[%q]", name)
		}
		for _, p := range alt {
			err := p.verify()
			if err != nil {
				return ctx.NewErrorf(nil, "%s: %s: %v", name, p.Directive, err)
			}
		}
	}
	return nil
}

// parse the given text using the named alternative
// check for unparsed text
func (this *Grammar) Parse(name string, text []byte) (any, error) {
	t0 := time.Now()
	p := pos{
		g:   this,
		src: text,
	}
	out, err := p.ConsumeAlt(this.alts[name])

	if err != nil {
		return out, err
	}

	if this.End != nil {
		p.IgnoreRE(this.End)
	}
	if p.Rem(10) != "" {
		return out, ctx.NewErrorf(nil, "unparsed: %q", p.Rem(80))
	}

	dt := time.Since(t0)
	this.Stats.ParseCt++
	this.Stats.ParseElapsed += dt
	return out, err
}
