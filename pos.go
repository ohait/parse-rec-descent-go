package parse

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/ohait/forego/ctx"
	"github.com/ohait/forego/enc"
)

type Pos struct {
	From int
	End  int
	File string
	Src  *Src
}

func (p Pos) P() Pos { return p }

func NewPos(filename string, in []byte) Pos {
	return Pos{
		File: filename,
		Src: &Src{
			bytes: in,
		},
	}
}

/*
func (this Pos) String() string {
	if this.File == "" {
		return fmt.Sprintf("%d-%d", this.From, this.End)
	}
	return fmt.Sprintf("%s:%d-%d", this.File, this.From, this.End)
}
*/

func (this Pos) GoString() string {
	return fmt.Sprintf("parse.Pos{%q:%s}", this.File, this.Extract(100))
}

func (this Pos) Extract(maxLines int) string {
	if this.Src == nil {
		return "<" + this.File + ">"
	}
	s := string(this.Src.bytes[this.From:this.End])
	if maxLines == 0 {
		return s
	}
	parts := strings.Split(s, "\n")
	if len(parts) > maxLines {
		parts = parts[0:maxLines]
		return strings.Join(parts, "\n")
	}
	return s
}

func (this Pos) String() string {
	line := this.Src.Line(this.From)
	if this.File == "" {
		return fmt.Sprintf("%d:%d-%d", line, this.From, this.End)
	}
	return fmt.Sprintf("%s:%d:%d-%d", this.File, line, this.From, this.End)
}

func (this Pos) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.String())
}

type Stats struct {
	Alternations    int // how many alternations were tested
	ParseTime       time.Duration
	BacktrackCount  int
	BacktrackAmount int // how many bytes were backtracked
}

type pos struct {
	g      *Grammar
	file   string
	src    *Src
	at     int
	end    int
	stack  []string
	commit bool // true if the current production is committed, used for errors
	p      *Prod
	stats  *Stats
}

func (this *pos) Log(f string, args ...any) {
	if this.g.Log != nil {
		rem := strings.TrimSuffix(fmt.Sprintf("%q", this.Rem(25))[1:], `"`)
		if len(rem) > 20 {
			rem = rem[0:20]
		}
		prod := ""
		if this.p != nil {
			prod = this.p.src
		}
		if i := strings.LastIndex(prod, "/"); i > 0 {
			prod = prod[i+1:]
		}
		this.g.Log("\033[0;33m%-020s\033[0;34m %s \033[0;35m%s\033[0m %s",
			rem,
			strings.Join(this.stack, "."),
			prod,
			fmt.Sprintf(f, args...),
		)
	}
}

func (this *pos) Rem(max int) string {
	rem := this.src.bytes[this.at:]
	if len(rem) > max {
		rem = rem[0:max]
	}
	return string(rem)
}

func (this *pos) IgnoreRE(re *regexp.Regexp, negative bool) error {
	m := re.Find(this.src.bytes[this.at:])
	if m == nil {
		if negative {
			return nil
		}
		//return ctx.NewErrorf(nil, "expected /%v/", re)
		return fmt.Errorf("❌ expected /%v/", re)
	}
	if negative {
		return fmt.Errorf("❌ unexpected /%v/", re)
	} else {
		this.at += len(m)
		if len(m) > 0 {
			this.Log("skip /%s/: %q", re, m)
		}
		return nil
	}
}

func (this *pos) ConsumeRE(re *regexp.Regexp, negative bool) (string, *Error) {
	m := re.FindIndex(this.src.bytes[this.at:])
	if m == nil {
		if negative {
			this.Log("✅ NEG AHEAD /%v/", re)
			return "", nil
		}
		this.Log("❌ FAIL /%v/", re)
		return "", this.NewErrorf("expected /%v/ got %q", re, this.Rem(80))
	}
	if m[0] != 0 {
		panic("re must match from the beginnin")
	}
	out := this.src.bytes[this.at : this.at+m[1]]
	if negative {
		//this.at += m[1]
		this.Log("❌ NEG AHEAD %q", out)
		return "", this.NewErrorf("unwanted /%v/", re)
	} else {
		this.at += m[1]
		this.Log("✅ CONSUMED /%v/ %q (%v)", re, out, m[1])
		return string(out), nil
	}
}

func (this *pos) push(n string) {
	this.stack = append(this.stack, n)
}
func (this *pos) pop() {
	this.stack = this.stack[0 : len(this.stack)-1]
}

// try to consume each of the alternatives in the given order
// first that succeed is returned
// if none succeed the first error is returned
func (this *pos) consumeProds(prods ...*Prod) (any, *Error) {
	this.stats.Alternations++
	switch len(prods) {
	case 0:
		panic("no alternatives") // Verify() woudl have catched this
	case 1:
		prod := prods[0]
		p := *this
		p.commit = false
		p.push("")
		p.Log("trying %s[%s] `%s`", prod.Name, prod.src, prod.Directive)
		out, err := prod.exec(&p)
		this.at = p.at
		return out, err
	default:
	}
	var errs []*Error
	for n, prod := range prods {
		p := *this
		p.commit = false
		p.push(fmt.Sprintf("%s/%d", prod.Name, n))
		p.Log("trying %s/%d[%s] `%s` ", prod.Name, n, prod.src, prod.Directive)
		out, err := prod.exec(&p)

		if err == nil {
			this.at = p.at
			return out, nil
		}
		if err.commit {
			p.Log("failed+commit %s[%s]: %v", prod.Name, prod.src, err)
			this.at = p.at
			return out, err
		}
		this.stats.BacktrackAmount += p.at - this.at
		this.stats.BacktrackCount++
		p.Log("failed %s[%s]: %v", prod.Name, prod.src, err)
		errs = append(errs, err)
	}
	this.Log("can't find any production")
	sort.Slice(errs, func(i, j int) bool {
		return errs[i].at > errs[j].at
	})
	for _, e := range errs {
		this.Log("» %v", e)
	}
	return nil, errs[0]
}

func (this *pos) NewErrorf(f string, args ...any) *Error {
	return &Error{
		err: fmt.Errorf(f, args...),
		//err: ctx.NewErrorf(nil, f, args...),
		at:     this.at,
		commit: this.commit,
	}
}

type Error struct {
	err    error
	at     int
	commit bool
}

var _ json.Marshaler = &Error{}
var _ enc.Marshaler = &Error{}

func (this Error) Error() string {
	return fmt.Sprintf("%v at %d", this.err, this.at)
}
func (this Error) Unwrap() error { return this.err }

func (this *Error) MarshalJSON() ([]byte, error) {
	return json.Marshal(this.Error())
}

func (this *Error) MarshalNode(c ctx.C) (enc.Node, error) {
	return enc.String(this.Error()), nil
}
