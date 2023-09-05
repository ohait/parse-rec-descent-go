package parse

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Aize-Public/forego/ctx"
)

type Pos struct {
	g     *Grammar
	src   []byte
	at    int
	stack []string
}

func (this *Pos) Log(f string, args ...any) {
	if this.g.Log != nil {
		this.g.Log("%-11q %s  %s",
			this.Rem(10),
			strings.Join(this.stack, " "),
			fmt.Sprintf(f, args...),
		)
	}
}

func (this *Pos) Rem(max int) string {
	rem := this.src[this.at:]
	if len(rem) > max {
		rem = rem[0:max]
	}
	return string(rem)
}

func (this *Pos) IgnoreRE(re *regexp.Regexp) error {
	m := re.Find(this.src[this.at:])
	if m == nil {
		return ctx.NewErrorf(nil, "expected /%v/", re)
	}
	this.at += len(m)
	if len(m) > 0 {
		this.Log("skip /%s/: %q ", re, m)
	}
	return nil
}

func (this *Pos) ConsumeRE(re *regexp.Regexp) (string, error) {
	m := re.FindIndex(this.src[this.at:])
	if m == nil {
		this.Log("❌ FAIL /%v/", re)
		return "", ctx.NewErrorf(nil, "expected /%v/ got %s", re, this.Rem(80))
	}
	if m[0] != 0 {
		return "", ctx.NewErrorf(nil, "regexp /%s/ doesn't match: %v", re, this.Rem(80))
	}
	out := this.src[this.at : this.at+m[1]]
	this.at += m[1]
	this.Log("✅ CONSUMED %q", out)
	return string(out), nil
}

func (this *Pos) push(n string) {
	this.stack = append(this.stack, n)
}
func (this *Pos) pop() {
	this.stack = this.stack[0 : len(this.stack)-1]
}

// try to consume each of the alternatives in the given order
// first that succeed is returned
// if none succeed the first error is returned
func (this *Pos) ConsumeAlt(alt Alt) (any, error) {
	if len(alt) == 0 {
		return nil, ctx.NewErrorf(nil, "no alternatives")
	}
	var errs []error
	start := this.at // checkpoint
	for n, prod := range alt {
		if len(alt) > 1 {
			this.push(fmt.Sprintf("%s/%d", prod.Name, n))
		} else {
			this.push(fmt.Sprintf("%s", prod.Name))
		}
		this.Log("trying `%s`", prod.Directive)
		out, err := prod.exec(this)
		if err == nil {
			this.pop()
			return out, nil
		}
		this.Log("failed <%s>: %v", prod.Name, err)
		this.at = start // backtrack
		errs = append(errs, err)
		this.pop()
	}
	this.Log("can't find any production")
	return nil, errs[0]
}
