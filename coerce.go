package parse

import (
	"reflect"

	"github.com/Aize-Public/forego/ctx"
)

// coerce the given value to the given type
// optionally creating new slices of the right type
func coerce(in reflect.Value, t reflect.Type) (reflect.Value, error) {
	if !in.IsValid() {
		return reflect.New(t).Elem(), nil // return zero value
	}
	if in.IsZero() {
		return reflect.New(t).Elem(), nil // return zero value
	}
	if in.Kind() == reflect.Interface {
		in = in.Elem()
	}
	if in.Type() == t {
		return in, nil
	}
	//log.Printf("coerce in: %v, t: %+v", in.Type(), t)
	if t.Kind() == reflect.Slice && in.Kind() == reflect.Slice {
		return coerceSlice(in, t)
	}
	if !in.CanConvert(t) {
		return in, ctx.NewErrorf(nil, "can't convert %v to %v", in.Type(), t)
	}
	in = in.Convert(t)
	return in, nil
}

func coerceSlice(in reflect.Value, t reflect.Type) (reflect.Value, error) {
	//log.Printf("coerceSlice(%v => %v)", in.Type(), t)
	out := reflect.MakeSlice(t, in.Len(), in.Len())
	for i := 0; i < in.Len(); i++ {
		v, err := coerce(in.Index(i), t.Elem())
		if err != nil {
			return in, err
		}
		out.Index(i).Set(v)
	}
	return out, nil
}
