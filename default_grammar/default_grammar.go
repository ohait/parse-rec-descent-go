// Package default_grammar provides a plug-and-play grammar for common use cases.
package default_grammar

import (
	"github.com/ohait/parse-rec-descent-go"
)

// New returns a Grammar with a default set of productions for arithmetic expressions.
func New() *parse.Grammar {
	g := parse.Grammar{}

	// Helper for left-associative operations
	leftAssoc := func(op any, tail []BinOp) any {
		for _, t := range tail {
			t.Left = op
			op = t
		}
		return op
	}

	// Helper for associativity tails
	assocTail := func(op string, right any, tail []BinOp) []BinOp {
		return append([]BinOp{{Op: op, Right: right}}, tail...)
	}

	// Define productions for addition and multiplication
	g.Alt("expr").Add(`term expr_`, leftAssoc)
	g.Alt("expr_").Add(`/[\+\-]/ term expr_`, assocTail)
	g.Alt("expr_").Add(``, nil)

	g.Alt("term").Add(`factor term_`, leftAssoc)
	g.Alt("term_").Add(`/[\*\/]/ factor term_`, assocTail)
	g.Alt("term_").Add(``, nil)

	g.Alt("factor").Add(`"(" expr ")"`, func(e any) any { return e })
	g.Alt("factor").Add(`/\d+/`, nil)

	return &g
}

// BinOp represents a binary operation.
type BinOp struct {
	Left  any    `json:"left"`
	Op    string `json:"op"`
	Right any    `json:"right"`
}