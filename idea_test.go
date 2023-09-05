package parse

import (
	"fmt"
	"testing"

	"github.com/Aize-Public/forego/test"
)

type Op interface {
}

type BinOp struct {
	Left  Op
	Op    string
	Right Op
}

func (this BinOp) String() string {
	return fmt.Sprintf("❲%v%s%v❳", this.Left, this.Op, this.Right)
}

type Lit struct {
	Val string // true, 123, 1.34, "foo"
}

func (this Lit) String() string {
	return fmt.Sprintf("%v", this.Val)
}

func TestManualAssoc(t *testing.T) {
	var g Grammar
	g.Add("add", `lit add_`, func(op Op, list []BinOp) (any, error) {
		for _, n := range list {
			n.Left = op
			op = n
		}
		return op, nil
	})
	g.Add("add_", `/[\+\-]/ lit add_`, func(op string, lit any, extra []BinOp) any {
		return append([]BinOp{{Op: op, Right: lit}}, extra...)
	})
	g.Add("add_", ``, func() any { return nil })

	g.Add("lit", `/\d+/`, func(v string) any { return Lit{v} })

	test.NoError(t, g.Verify())
	out, err := g.Parse("add", []byte(`1+2+3`))
	test.NoError(t, err)
	t.Logf("%+v", out)
}

func TestG(t *testing.T) {
	var g Grammar
	g.Add("expr", `cmp`, nil)
	g.Add("cmp", `add /(<|<=|==|>-|>|!=)/ add`, func(l Op, op string, r Op) BinOp {
		return BinOp{Left: l, Op: op, Right: r}
	})
	g.Add("cmp", `add`, nil)

	g.Add("add", `mul /[\+\-]/ add`, func(l Op, op string, r Op) BinOp {
		return BinOp{Left: l, Op: op, Right: r}
	})
	g.Add("add", `mul`, nil)
	g.Add("mul", `lit /[\*\/]/ mul`, func(l Op, op string, r Op) BinOp {
		return BinOp{Left: l, Op: op, Right: r}
	})
	g.Add("mul", `lit`, nil)
	g.Add("lit", `/\d+/`, func(v string) Lit { return Lit{v} })
	g.Add("lit", `/\d*\.\d*/`, func(v string) Lit { return Lit{v} })
	g.Add("lit", `/"([^"]|".)*"/`, func(v string) Lit { return Lit{v} })

	test.NoError(t, g.Verify())

	t.Logf("lit: %+v", g.alts["lit"][0])

	{
		out, err := g.Parse("expr", []byte("1+3*2"))
		test.NoError(t, err)
		t.Logf("%+v", out)
	}
	{
		out, err := g.Parse("expr", []byte("1+2+3"))
		test.NoError(t, err)
		t.Logf("%+v", out)
	}
}
