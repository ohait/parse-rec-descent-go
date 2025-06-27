package parse

import (
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"

	"github.com/ohait/forego/ctx/log"
)

type Alts struct {
	Grammar *Grammar
	Name    string
	prods   []*Prod
	retType reflect.Type
}

// Add a production to the given list
func (this *Alts) Add(directives string, fn any) *Prod {
	if this.Grammar.Log != nil {
		this.Grammar.Log("adding %s: %s", this.Name, directives)
	}
	if this.Grammar.alts == nil {
		this.Grammar.alts = map[string]*Alts{}
	}
	this.Grammar.Stats.Productions++
	_, file, line, _ := runtime.Caller(1)
	file = filepath.Base(file)
	p := &Prod{
		g:         this.Grammar,
		Name:      this.Name,
		Directive: directives,
		src:       fmt.Sprintf("%s:%d", file, line),
	}
	_, err := p.build("")
	if err != nil {
		log.Errorf(nil, "can't create prod %q: %v", this.Name, err)
		panic(err)
	}
	this.append(p)
	if fn == nil {
		return p
	}
	return p.Return(fn)
}

func (this *Alts) append(p *Prod) {
	this.prods = append(this.prods, p)
	if len(this.prods) == 1 {
		this.Grammar.Stats.Alternations++
	}
}
