package parse

import (
	"fmt"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Aize-Public/forego/ctx"
	"github.com/Aize-Public/forego/ctx/log"
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

	repCt atomic.Int32 // used to create internal names
}

func (this *Grammar) String() string {
	if this.Stats.ParseCt == 0 {
		return fmt.Sprintf("parse.Grammar{%d/%d}", this.Stats.Productions, this.Stats.Alternations)
	}
	return fmt.Sprintf("parse.Grammar{%d/%d %v}", this.Stats.Productions, this.Stats.Alternations, this.Stats.ParseElapsed/time.Duration(this.Stats.ParseCt))
}

func (this *Grammar) Dump() string {
	var keys []string
	for name := range this.alts {
		keys = append(keys, name)
	}
	sort.Strings(keys)
	var list []string
	for _, name := range keys {
		alt := this.alts[name]
		for _, p := range alt {
			list = append(list, fmt.Sprintf("%s: %s", name, p.Directive))
		}
	}
	return strings.Join(list, "\n") + "\n"
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
		this.alts = map[string]Alt{}
	}
	this.Stats.Productions++
	_, file, line, _ := runtime.Caller(1)
	file = filepath.Base(file)
	p := &Prod{
		g:         this,
		Name:      name,
		Directive: directives,
		src:       fmt.Sprintf("%s:%d", file, line),
	}
	_, err := p.build("")
	if err != nil {
		log.Errorf(nil, "can't create prod %q: %v", name, err)
		panic(err)
	}
	list := append(this.alts[name], p)
	this.alts[name] = list
	if len(list) == 1 {
		this.Stats.Alternations++
	}
	return p
}

// build the grammar, returns an error if the grammar is not complete
func (this *Grammar) Verify() error {
	for name, alt := range this.alts {
		//if this.Log != nil {
		//	this.Log("build[%q]", name)
		//}
		for _, p := range alt {
			err := p.verify()
			if err != nil {
				return ctx.NewErrorf(nil, "%s: %s: %v", name, p.Directive, err)
			}
		}
	}
	return nil
}

func (this *Grammar) Analyze() {
	w := map[string]float64{}
	names := []string{}
	for name, alt := range this.alts {
		names = append(names, name)
		w[name] = alt.weight(8)
	}
	lists.SortFunc(names, func(n string) float64 {
		return -w[n]
	})
	for _, name := range names {
		srcs := []string{}
		for _, p := range this.alts[name] {
			srcs = append(srcs, p.src)
		}
		this.Log("%q %.2f (%s)", name, w[name], strings.Join(srcs, " "))
	}
}

func (this *Grammar) Parse(prodName string, text []byte) (any, Stats, error) {
	return this.ParseFile(prodName, "", text)
}

// parse the given text using the named alternative
// check for unparsed text
func (this *Grammar) ParseFile(prodName string, fileName string, text []byte) (any, Stats, error) {
	var s Stats
	t0 := time.Now()
	p := pos{
		g:     this,
		file:  fileName,
		src:   text,
		stats: &s,
	}
	out, err := p.ConsumeAlt(this.alts[prodName])

	if err != nil {
		return out, s, err
	}

	if this.End != nil {
		p.IgnoreRE(this.End, false)
	}
	if p.Rem(10) != "" {
		return out, s, ctx.NewErrorf(nil, "unparsed: %q", p.Rem(80))
	}

	dt := time.Since(t0)
	this.Stats.ParseCt++
	this.Stats.ParseElapsed += dt
	s.ParseTime = dt
	return out, s, nil
}
