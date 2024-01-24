package parse

import (
	"strconv"
	"testing"

	"github.com/Aize-Public/forego/test"
)

func TestType(t *testing.T) {
	var g Grammar
	g.Alt("foo").Add(`bar /x/ + cuz`, func(bar, x string, cuz float64) string {
		return bar
	})
	g.Alt("bar").Add(`/\w+/`, func(s string) string {
		return s
	})
	g.Alt("cuz").Add(`/\d+/`, strconv.Atoi)

	test.Error(t, g.Verify())
}
