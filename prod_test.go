package parse

import (
	"fmt"
	"testing"

	"github.com/Aize-Public/forego/test"
)

func TestRegex(t *testing.T) {
	c := test.Context(t)
	{
		var g Grammar
		p := Prod{
			g:         &g,
			Directive: `/\S+/`,
		}
		_, err := p.build("")
		test.NoError(t, err)
		t.Logf("act: %+v", p.actions)
		test.EqualsGo(t, 1, len(p.actions))
		pos := pos{
			g:   &g,
			src: &Src{bytes: []byte("foo bar")},
		}
		out, err := p.actions[0].exec(&pos)
		test.NoError(t, err)
		test.EqualsJSON(c, `foo`, out)
	}
}

func TestText(t *testing.T) {
	c := test.Context(t)
	{
		var g Grammar
		p := Prod{
			g:         &g,
			Directive: `/\S+/`,
		}
		_, err := p.build("")
		test.NoError(t, err)
		t.Logf("act: %+v", p.actions)
		test.EqualsGo(t, 1, len(p.actions))
		pos := pos{
			g:   &g,
			src: &Src{bytes: []byte("foo bar")},
		}
		out, err := p.actions[0].exec(&pos)
		test.NoError(t, err)
		test.EqualsJSON(c, `foo`, out)
	}
}

func TestDescent(t *testing.T) {
	{
		g := Grammar{
			Log: t.Logf,
			End: Whitespaces,
		}
		g.Alt("main").Add(`word word`, nil)
		g.Alt("word").Add(`/\w+/`, nil).WS = Whitespaces
		out, _, err := g.Parse("main", []byte(" foo\t\nbar\n"))
		test.NoError(t, err)
		test.EqualsGo(t, []any{"foo", "bar"}, out)
	}
}

func TestDirectiveParsing(t *testing.T) {
	{
		p := Prod{
			Directive: "/a/ /cd/",
		}
		_, err := p.build("")
		test.NoError(t, err)
		t.Logf("act: %+v", p.actions)
		test.EqualsGo(t, 2, len(p.actions))
	}
	{
		p := Prod{
			Directive: `/a\/b/`,
		}
		t.Logf("in: `%s`", p.Directive)
		_, err := p.build("")
		test.NoError(t, err)
		t.Logf("act: %+v", p.actions)
		test.EqualsGo(t, 1, len(p.actions))
	}
}

func TestNoAction(t *testing.T) {
	{
		var g Grammar
		g.Add("main", `/a+/`)
		out, _, err := g.Parse("main", []byte("aa"))
		test.NoError(t, err)
		test.EqualsGo(t, `aa`, out)
	}
	{
		var g Grammar
		g.Add("main", ``)
		out, _, err := g.Parse("main", []byte(""))
		test.NoError(t, err)
		test.EqualsGo(t, nil, out)
	}
	{
		var g Grammar
		g.Add("main", `word word`)
		g.Add("word", `/\w+/`).WS = Whitespaces
		out, _, err := g.Parse("main", []byte("xyz foo"))
		test.NoError(t, err)
		test.EqualsGo(t, []any{"xyz", "foo"}, out)
	}
}

func TestSpace(t *testing.T) {
	c := test.Context(t)
	{
		g := Grammar{
			End: Whitespaces,
		}
		g.Add("main", `word word`)
		g.Add("word", `/\w+/`).WS = Whitespaces
		out, _, err := g.Parse("main", []byte("  foo\n\tfoo\n"))
		test.NoError(t, err)
		test.EqualsJSON(c, []any{"foo", "foo"}, out)
	}
	{
		g := Grammar{
			End: CommentsAndWhitespaces,
			Log: t.Logf,
		}
		g.Add("main", `words`)
		g.Add("words", `word words`).Return(func(l string, tail []string) []string { return append([]string{l}, tail...) })
		g.Add("words", ``).Return(func() []string { return nil })
		g.Add("word", `/\w+/`).WS = CommentsAndWhitespaces
		out, _, err := g.Parse("main", []byte("1 // ignore\n\t2\n 3//"))
		test.NoError(t, err)
		test.EqualsJSON(c, []any{"1", "2", "3"}, out)
	}
}

func TestBaseAction(t *testing.T) {
	{
		g := Grammar{
			Log: t.Logf,
		}
		g.Add("main", `word word`).Return(func(left, right string) (string, error) {
			if left == right {
				return left, nil
			}
			return "", fmt.Errorf("expected same word, got %q and %q", left, right)
		})
		g.Add("word", `/\w+/`).WS = Whitespaces
		out, _, err := g.Parse("main", []byte("foo foo"))
		test.NoError(t, err)
		test.EqualsGo(t, "foo", out)
	}
	{
		var g Grammar
		g.Add("main", `word word`).Return(func(left, right int) int {
			t.Logf("%d+%d", left, right)
			return right + left
		})
		g.Add("word", `/\w+/`).Return(func(p Pos, s string) int {
			t.Logf("%q => %d - %d", s, p.End, p.From)
			return p.End - p.From
		}).WS = Whitespaces
		out, _, err := g.Parse("main", []byte("xyz foobar"))
		test.NoError(t, err)
		test.EqualsGo(t, 10, out)
	}
}

func TestCommit(t *testing.T) {
	{
		var g Grammar
		g.Log = t.Logf
		g.Add("op", `parens`)
		g.Add("op", ``).Return(func() any {
			t.Fatalf("commit not honored 1 level above")
			return nil
		})
		g.Add("parens", `"(" + word ")"`)
		g.Add("parens", ``).Return(func() any {
			t.Fatalf("commit not honored")
			return nil
		})
		g.Add("word", `/\w+/`)
		_, _, err := g.Parse("op", []byte(`(foobar`))
		test.Contains(t, err.Error(), ")")
	}
}

func TestNegative(t *testing.T) {
	{
		var g Grammar
		g.Log = t.Logf
		g.Add("add", `word "+" !"a" + word`)
		g.Add("word", `/\w+/`)
		{
			_, _, err := g.Parse("add", []byte(`a+abc`))
			test.Contains(t, err.Error(), "a")
		}
		{
			_, _, err := g.Parse("add", []byte(`a+bcd`))
			test.NoError(t, err)
		}
	}
}

func TestAssocSimple(t *testing.T) {
	c := test.Context(t)
	var g Grammar
	g.Log = t.Logf
	g.Add("ident", `/[a-zA-Z]\w+/`).Return(func(s string) string {
		return s
	}).WS = Whitespaces
	g.Add("list", `"list:" ident(s)`).Return(func(list []string) []string {
		return list
	})
	t.Logf("%s", g.Dump())
	out, _, err := g.Parse("list", []byte(`list: adam john`))
	test.NoError(t, err)
	test.EqualsJSON(c, `["adam","john"]`, out)
}

func TestAssocSep(t *testing.T) {
	c := test.Context(t)
	var g Grammar
	g.Log = t.Logf
	g.Add("ident", `/[a-zA-Z]\w+/`).Return(func(s string) string {
		return s
	}).WS = Whitespaces
	g.Add("list", `"list:" ident(s ",")`).Return(func(list []string) []string {
		return list
	}).WS = Whitespaces
	t.Logf("%s", g.Dump())
	out, _, err := g.Parse("list", []byte(`list: adam, john ,luke`))
	test.NoError(t, err)
	test.EqualsJSON(c, `["adam","john","luke"]`, out)
}
