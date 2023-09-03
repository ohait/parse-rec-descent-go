package parse

import (
	"log"
	"reflect"
)

type Alt []*Prod

type Prod struct {
	G         *Grammar
	Name      string
	Directive string
	act       func(p *Pos) ([]any, error)
	ret       func(in any) (any, error)
	retType   reflect.Type
}

func (this *Prod) exec(in []any, p *Pos) (any, error) {
	out, err := this.act(p)
	if err != nil {
		return nil, err
	}
	if this.ret == nil {
		return out, err
	} else {
		log.Printf("ret(%v, %v)", in, out)
		//out, err := this.ret(append(in, out...))
		out, err := this.ret(out)
		log.Printf("ret() => %v", out)
		return out, err
	}
}

func (this *Prod) Return(fn any) {
	switch action := fn.(type) {
	case func(in any) (any, error):
		this.ret = action
	case func(in any) any:
		this.ret = func(in any) (any, error) {
			return action(in), nil
		}
	case func(in []any) (any, error):
		this.ret = func(in any) (any, error) {
			return action(in.([]any))
		}
	case func(in []any) any:
		this.ret = func(in any) (any, error) {
			return action(in.([]any)), nil
		}
	case nil:
		return
	default:
		f := reflect.ValueOf(action)
		t := f.Type()
		this.ret = func(in any) (any, error) {
			log.Printf("calling %v with %v", t, in)
			var list []reflect.Value
			for i, in := range in.([]any) {
				t := t.In(i) // expected type
				v := coerce(reflect.ValueOf(in), t)
				log.Printf("ARG: %v", v)
				list = append(list, v)
			}
			this.retType = t.Out(0)
			//log.Printf("CALL[%v](%v)", f, list)
			out := f.Call(list)
			switch len(out) {
			case 1:
				return out[0].Interface(), nil
			case 2:
				first := out[0].Interface()
				second := out[1].Interface()
				if second == nil {
					return first, nil
				}
				log.Printf("ERR: %T (%+v)", second, second)
				return first, second.(error)
			default:
				panic("can only return (any) or (any, error)")
			}
		}
		//panic(fmt.Sprintf("unsupported action: %T", fn))
	}
}
