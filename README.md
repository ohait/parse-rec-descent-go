# parse-rec-descent-go
A runtime recursive descent parser inspired by Parse::RecDescent but in Go.

It allows for building parsers dynamically, with a quasi-BNF syntax.

## Features
- Define grammars programmatically in Go.
- Supports alternations, repetitions, and custom return functions.
- Built-in support for left/right associativity and precedence.
- Default grammar for plug-and-play usage (see below).

## Quick Start

### Basic Example
```go
import parse "github.com/ohait/parse-rec-descent-go"

var g parse.Grammar
g.Alt("add").Add(`lit add_`, func(op Op, list []BinOp) (any, error) {
    for _, n := range list {
        n.Left = op
        op = n
    }
    return op, nil
})

g.Alt("add_").Add(`/[\+\-]/ lit add_`, func(op string, lit any, tail []BinOp) any {
    return append([]BinOp{{Op: op, Right: lit}}, tail...)
})
g.Alt("add_").Add(``, nil)

g.Alt("lit").Add(`/\d+/`, func(v string) int {
    return strconv.Atoi(v)
})

err := g.Verify() // Ensure no production links to empty ones

out, _, err := g.Parse("add", "", []byte("1+2+3")) // Returns BinOp{BinOp{1, "+", 2}, "+", 3}
```

### Default Grammar
A default grammar is provided for common use cases. Import and use it like this:
```go
import "github.com/ohait/parse-rec-descent-go/default_grammar"

g := default_grammar.New()
ast, _, err := g.Parse("expr", "", []byte("1+2*3"))
```

## How It Works
You define productions and alternations. The parser descends through the grammar, trying to match the input text.

### Productions and Alternations
```go
var g parse.Grammar
g.Alt("my_prod").Add(`/(\+|-)/ ident`, nil)
g.Alt("ident").Add(`/[+\-]?\d*/`, nil) // integer
g.Alt("ident").Add(`/(true|false)/`, nil) // bool
```

### Return Functions
When a production matches, you can define a function to process the results:
```go
g.Alt("x").Add(`a list`, func(a A, list []X) (X, error) {
    return X{a, list}
})
```

### Associativity and Precedence
For expressions like `1+2*3`, define nested productions:
```go
g.Alt("add").Add(`mul add_`, leftAssoc)
g.Alt("add_").Add(`/[\+\-]/ mul add_`, assocTail)
g.Alt("add_").Add(``, nil)

g.Alt("mul").Add(`num mul_`, leftAssoc)
g.Alt("mul_").Add(`/[\*\/]/ num mul_`, assocTail)
g.Alt("mul_").Add(``, nil)

g.Alt("num").Add(`/\d+/`, nil)
```

### Error Handling and Commit
Use `+` to commit to a production and avoid backtracking:
```go
g.Alt("my_prod").Add(`a + b`, nil) // If 'a' matches, 'b' must match
```

### Negative Look-Ahead
Prefix a directive with `!` to match only if it does *not* appear:
```go
g.Alt("my_prod").Add(`!"forbidden" a`, nil)
```

## Default Grammar
The `default_grammar` package provides a ready-to-use grammar for arithmetic expressions, including:
- Basic arithmetic (`+`, `-`, `*`, `/`)
- Parentheses for grouping
- Left associativity and precedence rules

## Advanced Usage
### Custom Actions
```go
g.Add("expr", `term expr_`).Return(func(op any, tail []BinOp) any {
    for _, t := range tail {
        t.Left = op
        op = t
    }
    return op
})
```

### Debugging
Set `g.Log = func(format string, args ...interface{})` to enable debug output.

## License
MIT
