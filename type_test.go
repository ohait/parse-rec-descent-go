package parse

import (
	"strconv"
	"testing"

	"github.com/Aize-Public/forego/test"
)

func TestType(t *testing.T) {
	var g Grammar
	g.Alt("foo").Add(`bar cuz`, func(bar, cuz string) string {
		return bar + " " + cuz
	})
	g.Alt("bar").Add(`/\w+/`, func(s string) string {
		return s
	})
	g.Alt("cuz").Add(`/\d+/`, strconv.Atoi)

	test.Error(t, g.Verify())
}
