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

