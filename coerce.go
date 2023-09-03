package parse

import (
	"fmt"
	"log"
	"reflect"
)

// coerce the given value to the given type
// optionally creating new slices of the right type
func coerce(in reflect.Value, t reflect.Type) reflect.Value {
	if !in.IsValid() {
		return reflect.New(t).Elem() // return zero value
	}
	if in.IsZero() {
		return reflect.New(t).Elem() // return zero value
	}
	if in.Kind() == reflect.Interface {
		in = in.Elem()
	}
	log.Printf("coerce in: %v, t: %+v", in.Type(), t)
	if t.Kind() == reflect.Slice && in.Kind() == reflect.Slice {
		return coerceSlice(in, t)
	}
	if !in.CanConvert(t) {
		panic(fmt.Sprintf("can't convert %v to %v", in.Type(), t))
	}
	in = in.Convert(t)
	return in
}

func coerceSlice(in reflect.Value, t reflect.Type) reflect.Value {
	out := reflect.MakeSlice(t, in.Len(), in.Len())
	for i := 0; i < in.Len(); i++ {
		v := coerce(in.Index(i), t.Elem())
		out.Index(i).Set(v)
	}
	return out
}
