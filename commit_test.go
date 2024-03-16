package parse_test

import (
	"testing"

	"github.com/ohait/forego/test"
	"github.com/ohait/parse-rec-descent-go"
)

func TestCommitErr(t *testing.T) {
	var g parse.Grammar
	if testing.Verbose() {
		g.Log = t.Logf
	}
	g.Add("expr", `mul`)

	g.Add("mul", `add mul_`)
	g.Add("mul_", `"*"+ add mul_`)
	g.Add("mul_", ``)

	g.Add("add", `num add_`)
	g.Add("add_", `"+"+ num add_`)
	g.Add("add_", ``)

	g.Add("num", `/\d+/`)

	_, _, err := g.Parse("expr", []byte(`2+3+`))
	test.Error(t, err)
	test.Contains(t, err.Error(), "expected num got")
}
