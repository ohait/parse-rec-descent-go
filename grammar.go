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
	"github.com/Aize-Public/forego/utils/lists"
	"github.com/Aize-Public/forego/utils/maps"
)

type Grammar struct {

	// Trailing regexp
	End *regexp.Regexp

	alts map[string]*Alts
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
		for _, p := range alt.prods {
			list = append(list, fmt.Sprintf("%s: %s", name, p.Directive))
		}
	}
	return strings.Join(list, "\n") + "\n"
}

var Whitespaces = regexp.MustCompile(`[\s\n\r]*`)
var CommentsAndWhitespaces = regexp.MustCompile(`(\s|//[^\n]*\n?)*`)

// return the productions for the given name (can be empty)
func (this *Grammar) Alt(name string) *Alts {
	if this.alts == nil {
		this.alts = map[string]*Alts{}
	}
	a := this.alts[name]
	if a == nil {
		a = &Alts{
			Grammar: this,
			Name:    name,
		}
		this.alts[name] = a
	}
	return a
}

// Add a new production with the given name and directive
// if many elements are in the directive, it returns a list of objects
// otherwise return the only element
// returns a production that can further be tweaked, adding a Return() action which override the above, and changing the whitespace
// panics if anything is wrong (you normally don't want to handle the error, since can be seen as a compile time error)

// deprecated: use Grammar{}.Alt(name).Add(directive, func(...))
func (this *Grammar) Add(name string, directives string, extra ...any) *Prod {
	if this.Log != nil {
		this.Log("adding %s: %s", name, directives)
	}
	if this.alts == nil {
		this.alts = map[string]*Alts{}
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
	this.Alt(name).append(p)
	switch len(extra) {
	case 0:
		/* DO NOT CHECK
		switch len(p.actions) {
		case 0:
			p.retType = reflect.TypeOf(nil)
		case 1:
			p.retType = p.actions[0].p.retType
		default:
			p.retType = reflect.TypeOf([]any{})
		}
		*/
	case 1:
		return p.Return(extra[0])
	default:
		panic("unsupported")
	}
	return p
}

// build the grammar, returns an error if the grammar is not complete
func (this *Grammar) Verify() error {
	for name, alt := range this.alts {
		//if this.Log != nil {
		//	this.Log("build[%q]", name)
		//}
		for _, p := range alt.prods {
			err := p.verify()
			if err != nil {
				return ctx.NewErrorf(nil, "%s: %s: %v", name, p.Directive, err)
			}
		}
	}
	return nil
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
		src:   &Src{bytes: text},
		stats: &s,
	}
	alt := this.alts[prodName]
	if alt == nil {
		return nil, s, ctx.NewErrorf(nil, "no prod named %q", prodName)
	}
	out, err := p.consumeProds(alt.prods...)

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

func (this *Grammar) Analyze() {
	for _, alt := range this.alts {
		alt.cost = 0.0
	}
	for i := 0; i < 8; i++ {
		for _, alt := range this.alts {
			w := 0.0
			for _, p := range alt.prods {
				for _, a := range p.actions {
					if a.commit {
						break
					}
					if a.prod != "" {
						pw := 0.8 * this.alts[a.prod].cost
						if pw < 0.1 {
							pw = 0.1
						}
						w += pw
					} else {
						w += 1 // cost 1 for simple regex
					}
				}
			}
			alt.cost = w
		}
	}
	alts := maps.Pairs(this.alts).Values()
	lists.SortFunc(alts, func(a *Alts) float64 {
		return -a.cost
	})
	if len(alts) > 10 {
		alts = alts[10:]
	}
	for _, a := range alts {
		log.Infof(nil, "%q %.3f", a.Name, a.cost)
	}
}
