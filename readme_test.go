package parse_test

import (
	"testing"

	"github.com/Aize-Public/forego/test"
	"github.com/ohait/parse-rec-descent-go"
)

func TestReadme1(t *testing.T) {
	var g parse.Grammar
	g.Log = t.Logf
	g.Add("div", `num div_`)      // return [num, [...]
	g.Add("div_", `"/" num div_`) // return ["/", num, [...]]
	g.Add("div_", ``)             // return nil
	g.Add("num", `/\d+/`)

	ast, err := g.Parse("div", []byte(`40/10/2`))
	t.Logf("%+v (%v)", ast, err)
	test.NoError(t, err)
}

func TestReadme2(t *testing.T) {
	type BinOp struct {
		Left  any
		Op    string
		Right any
	}
	var g parse.Grammar
	g.Log = t.Logf
	g.Add("div", `num div_`).Return(func(op any, tail []BinOp) any {
		for _, e := range tail {
			e.Left = op
			op = e
		}
		return op
	})
	g.Add("div_", `/\// num div_`).Return(func(op string, num any, tail []BinOp) []BinOp {
		return append([]BinOp{{Op: op, Right: num}}, tail...)
	})
	g.Add("div_", ``) // return nil
	g.Add("num", `/\d+/`)

	ast, err := g.Parse("div", []byte(`40/10/2`))
	t.Logf("%+v (%v)", ast, err)
	test.NoError(t, err)
}

func TestReadme3(t *testing.T) {
	type BinOp struct {
		Left  any
		Op    string
		Right any
	}
	type Parens struct {
		Op any
	}
	var g parse.Grammar
	//g.Log = t.Logf

	leftAssoc := func(op any, tail []BinOp) any {
		for _, t := range tail {
			t.Left = op
			op = t
		}
		return op
	}
	assocTail := func(op string, right any, tail []BinOp) []BinOp {
		return append([]BinOp{{Op: op, Right: right}}, tail...)
	}

	g.Add("add", `mul add_`).Return(leftAssoc)
	g.Add("add_", `/(\+|\-)/ mul add_`).Return(assocTail).WS = parse.Whitespaces
	g.Add("add_", ``)

	g.Add("mul", `unary mul_`).Return(leftAssoc)
	g.Add("mul_", `/(\*|\/|%)/ + unary mul_`).Return(assocTail).WS = parse.CommentsAndWhitespaces
	g.Add("mul_", ``)

	g.Add("unary", `"-" op`).Return(func(op any) BinOp {
		return BinOp{Op: "-", Right: op}
	})
	g.Add("unary", `op`)

	g.Add("op", `"(" + add ")"`).Return(func(op any) Parens {
		return Parens{op}
	})
	g.Add("op", `/\d+/`)

	ast, err := g.Parse("add", []byte(`1+2*(3+4)`))
	t.Logf("%#v (%v)", ast, err)
	test.NoError(t, err)
}
