# parse-rec-descent-go
A runtime recursive descendant parser inspired by Parse::RecDescent but in Go

It allows for building parser dynamically, with a quasi-BNF syntax:

```go
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

err := g.Build()

out, err := g.Parse("add", []byte("1+2+3")) // returns ❲❲1+2❳+3❳
```

