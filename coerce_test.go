package parse

import (
	"reflect"
	"testing"

	"github.com/ohait/forego/test"
)

func TestCoerce(t *testing.T) {
	s := "str"

	v, err := coerce(reflect.ValueOf(s), reflect.TypeOf(s))
	test.NoError(t, err)
	test.EqualsGo(t, reflect.TypeOf(s), v.Type())

	{
		s := []any{"1", "2"}
		out := []string{}
		v, err := coerce(reflect.ValueOf(s), reflect.TypeOf(out))
		test.NoError(t, err)
		test.EqualsGo(t, reflect.TypeOf(out), v.Type())
	}
}
