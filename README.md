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


## Return

When a production matches, it generates a list of matching arguments.

By default, if the list is empty, `nil` is returned.
If only 1 element is present, the element itself is returned.
otherwise the list is returned.

Users can modify this behaviour by setting a custom function.

The function will be called when productions succeed, and special `reflect` magic happens under the hood to call the function, so to map some of the
directive to the arguments.

```go
g.Add("x", `a list`).Return(func(a A, list []X) (X, error) {
  return X{a, list}
})
g.Add("a", `x`).Return(func(p parse.Pos, x X) A {
  return Y{p, x}
})
```

In the example above, you can see how each directive has matching argument, and the type is converted for you.

In the second production, you can see how you can add an extra argument (only the first one) which accept a `parse.Pos` which can be useful for debugging
or error reporting

## commit 

The special directive `!` an be added in the middle of a production to specify that no further alternatives must be tried.

This both reduce the backtracking but also improves the error reporting


## error

Every time a production fails, it returns an error that reports how fair it went parsing the input.

When several alternations fails, the error that parsed the most text is used.

## Expression example

### Associativity

```
   40
   40/10
   40/10/2
```

Above there are 3 examples of expressions, if you have to parse that you need to be able to handle associativity, and in this case left associativity.

Note: Why is important to associate to the left? Consider the last example, if you associate to the left, you will execute `(40/10)/2` which returns `2`, but if you associate to the right you would `40/(10/2)` which returns `8`.

To parse the example above, you could use the simple grammar:

```go
  var g Grammar
  g.Add("div", `num "/" num`)
  g.Add("div", `num`)
  g.Add("num", `/\d+/`)
```

This simple example will do the job, but has the side effect be parse multiple time the left side of any `/`. In this case it won't matter much, but if the left side is a very complex expression, all the inefficiency would pile up to a crawl.

A better approach is to write a grammar like this:

```go
  var g Grammar
  g.Add("div", `num div_`) // return [num, [...]
  g.Add("div_", `"/" num div_`) // return ["/", num, [...]]
  g.Add("div_", ``) // return nil
  g.Add("num", `/\d+/`)
```

While slightly more complicated, it avoids completely backtracking, and the left side is always parsed only once.

### Action

With a simple grammar like the one above, it easy to make sense of the output tree:
```go
  ast, out := g.Parse("div", []byte("40/10/2"))
  // ast => [40 [10 [2 <nil>]]]
```

But with increasingly complex grammars, it's best to generate custom objects as output:

```go
    g.Add("div", `num div_`).Return(func(op any, tail []BinOp) any {
        // loop thru the tail and associate left
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
```

The above code will do a better job at generating a useful AST:
```go
  BinOp{
    Left:BinOp{
      Left:"40", 
      Op:"/",
      Right:"10",
    },
    Op:"/",
    Right:"2",
  }
```

### Precedence

Building from the above example, if we want to add all the 4 basic math operators, we need to consider precedence.

```
   1+2*3+4
```

The above example should be parsed as `1+(2*3)+4` and gives `11`, as opposed to `((1+2)*3)+4` which would return `13`.

The best way to achieve this is to build a grammar that expect expressions nested into each other:

```go
  var g Grammar

  g.Add("add", `mul add_`)
  g.Add("add_", `/[\+\-]/ mul add_`)
  g.Add("add_", ``)

  g.Add("mul", `num mul_`)
  g.Add("mul_", `/[\*\/]/ num mul_`)
  g.Add("mul_", ``)

  g.Add("num", `/\d+/`)
```

By nesting `mul` into `add`, we implicitly enforce higher precedence, because when the parser try to capture multiplication it will happen "earlier"

### Recursion

Expanding from the examples above, we could add parenthesis and proper `Return()` actions:

```go
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
	g.Add("mul_", `/(\*|\/|%)/ ! unary mul_`).Return(assocTail).WS = parse.CommentsAndWhitespaces
	g.Add("mul_", ``)

	g.Add("unary", `"-" op`).Return(func(op any) BinOp {
		return BinOp{Op: "-", Right: op}
	})
	g.Add("unary", `op`)

	g.Add("op", `"(" ! add ")"`).Return(func(op any) Parens {
		return Parens{op}
	})
	g.Add("op", `/\d+/`)
```

Which can parse complex expressions like:

```go
	ast, err := g.Parse("add", []byte(`1+2*(3+4)`))

  BinOp{
    Left:"1",
    Op:"+",
    Right:BinOp{
      Left:"2",
      Op:"*",
      Right:Parens{
        Op:BinOp{
          Left:"3",
          Op:"+",
          Right:"4",
        },
      },
    },
  }
```

