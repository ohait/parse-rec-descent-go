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
	g.Add("add", `lit add_`).Return(func(op Op, list []BinOp) (any, error) {
		for _, n := range list {
			n.Left = op
			op = n
		}
		return op, nil
	})
	g.Add("add_", `/[\+\-]/ lit add_`).Return(func(op string, lit any, extra []BinOp) any {
		return append([]BinOp{{Op: op, Right: lit}}, extra...)
	})
	g.Add("add_", ``).Return(func() any { return nil })

	g.Add("lit", `/\d+/`).Return(func(v string) any { return Lit{v} })

	test.NoError(t, g.Verify())
	out, err := g.Parse("add", []byte(`1+2+3`))
	test.NoError(t, err)
	t.Logf("%+v", out)
}

func TestGrammar(t *testing.T) {
	var g Grammar
	g.Log = t.Logf
	g.Add("expr", `cmp`)
	g.Add("cmp", `add /(<|<=|==|>-|>|!=)/ add`).Return(func(l Op, op string, r Op) BinOp {
		return BinOp{Left: l, Op: op, Right: r}
	})
	g.Add("cmp", `add`)

	g.Add("add", `mul /[\+\-]/ add`).Return(func(l Op, op string, r Op) BinOp {
		return BinOp{Left: l, Op: op, Right: r}
	})
	g.Add("add", `mul`)
	g.Add("mul", `lit /[\*\/]/ mul`).Return(func(l Op, op string, r Op) BinOp {
		return BinOp{Left: l, Op: op, Right: r}
	})
	g.Add("mul", `lit`)
	g.Add("lit", `/\d+/`).Return(func(v string) Lit { return Lit{v} })
	g.Add("lit", `/\d*\.\d*/`).Return(func(v string) Lit { return Lit{v} })
	g.Add("lit", `/"([^"]|".)*"/`).Return(func(v string) Lit { return Lit{v} })

	test.NoError(t, g.Verify())

	t.Logf("lit: %+v", g.alts["lit"][0])

	{
		out, err := g.Parse("expr", []byte("1+2"))
		test.NoError(t, err)
		t.Logf("%+v", out)
	}
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
