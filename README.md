# parse-rec-descent-go
A runtime recursive descendant parser inspired by Parse::RecDescent but in Go

It allows for building parser dynamically, with a quasi-BNF syntax:

```go
var g parse.Grammar
g.Add("add", `lit add_`).Return(
  func(op Op, list []BinOp) (any, error) {
  	for _, n := range list {
  		n.Left = op
  		op = n
  	}
  	return op, nil
  })

g.Add("add_", `/[\+\-]/ lit add_`).Return(
  func(op string, lit any, tail []BinOp) any {
  	return append([]BinOp{{Op: op, Right: lit}}, tail...)
  })
g.Add("add_", ``)

g.Add("lit", `/\d+/`).Return(
  func(v string) int {
    return strconv.Atoi(v)
  })

err := g.Verify() // make sure no production link to empty ones

out, err := g.Parse("add", []byte("1+2+3")) // returns BinOp{BinOp{1, "+", 2}, "+", 3}
```

## How it works

You define some production, and then ask the grammar to parse some text.

The parser will then *descend* the production directives and try to consume all the text.

Each production can return custom structures, which allow you to generate AST directly where you define the grammar.

## Production and Alternation

```go
  var g Grammar
  g.Add("my_prod", `/(\+|-)/ ident`)
  g.Add("ident", `/[+\-]?\d*/`) // integer
  g.Add("ident", `/(true|false)/`) // bool
```

A grammar is made of several productions, each production has a name and a series of directives.

If all the directives parse the input correctly, the production matches.

If multiple productions have the same name, they are part of an alternation and they are checked in the sequence they were added to the grammar.

First one that succeed will be used, and if none succeed the whole alternation fails.

Each time another production of the same alternation is checked, the position of the parser is restored.

