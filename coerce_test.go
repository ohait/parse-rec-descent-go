package parse

import (
	"reflect"
	"testing"

	"github.com/Aize-Public/forego/test"
)

func TestCoerce(t *testing.T) {
	s := "str"

	v := coerce(reflect.ValueOf(s), reflect.TypeOf(s))
	test.EqualsGo(t, reflect.TypeOf(s), v.Type())

	{
		s := []any{"1", "2"}
		out := []string{}
		v := coerce(reflect.ValueOf(s), reflect.TypeOf(out))
		test.EqualsGo(t, reflect.TypeOf(out), v.Type())
	}
}
