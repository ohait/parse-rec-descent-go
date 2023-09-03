package parse

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/Aize-Public/forego/ctx"
)

type Pos struct {
	G     *Grammar
	src   []byte
	at    int
	stack []string
}

func (this *Pos) Log(f string, args ...any) {
	log.Printf("%s| %s",
		strings.Join(this.stack, " "),
		fmt.Sprintf(f, args...),
	)
}

func (this *Pos) Rem(max int) string {
	rem := this.src[this.at:]
	if len(rem) > max {
		rem = rem[0:max]
	}
	return string(rem)
}

func (this *Pos) ConsumeRE(re *regexp.Regexp) (string, error) {
	m := re.FindIndex(this.src[this.at:])
	if m == nil {
		this.Log("FAIL /%v/ at %s", re, this.Rem(30))
		return "", ctx.NewErrorf(nil, "expected /%v/ got %s", re, this.Rem(80))
	}
	if m[0] != 0 {
		return "", ctx.NewErrorf(nil, "regexp doesn't match from start: %v", re)
	}
	out := this.src[this.at : this.at+m[1]]
	this.at += m[1]
	this.Log("CONSUMED %q, rem %q", out, this.Rem(30))
	return string(out), nil
}

func (this *Pos) push(n string) {
	this.stack = append(this.stack, n)
}
func (this *Pos) pop() {
	this.stack = this.stack[0 : len(this.stack)-1]
}

func (this *Pos) ConsumeAlt(in []any, alt Alt) (any, error) {
	var errs []error
	start := this.at
	for n, prod := range alt {
		this.push(fmt.Sprintf("%s%d", prod.Name, n))
		this.Log("trying <%s> `%s` %q (arg: %v)", prod.Name, prod.Directive, this.Rem(30), in)
		out, err := prod.exec(in, this)
		if err == nil {
			this.pop()
			return out, nil
		}
		this.at = start // backtrack
		errs = append(errs, err)
		this.pop()
	}
	return nil, errs[0]
}
