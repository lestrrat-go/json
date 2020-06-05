package json

import (
	"bytes"
	stdlib "encoding/json"
	"fmt"
	"reflect"
	"sync"

	"github.com/pkg/errors"
)

var zeroval reflect.Value

type Context interface {
	// Bool assigns the value pointed by the Context to the specified
	// destination, which must be a pointer to a variable compatible
	// with bool.
	// If the underlying value is not a boolean, an error will be returned
	Bool(interface{}) error

	// Float assigns the value pointed by the Context to the specified
	// destination, which must be a pointer to a variable compatible
	// with float64.
	// If the underlying value is not a floating point number, an error will be returned
	Float(interface{}) error

	// Int assigns the value pointed by the Context to the specified
	// destination, which must be a pointer to a variable compatible
	// with int64.
	// If the underlying value is not an integer, an error will be returned
	Int(interface{}) error

	// Index returns a new JSON Context pointing to the value
	// of the element at the specified index of the array
	// For example, given a JSON array `{"one", 2, true}`, you can
	// get the Context pointing to the second element by calling `j.Index(1)`
	//
	// When an error is found, the returned Context is an invalid,
	// and calling methods on it will only return the original error
	Index(int) Context

	// Map returns the value as a Go map. If the underlying
	// value is not a JSON object, then an error along with
	// a nil value is returned.
	Map(interface{}) error

	// MapIndex returns a new JSON Context pointing to the value
	// of the named field in the map
	// For example, given a JSON object `{"foo": "bar"}`, you can
	// get the Context pointing to `"bar"` by calling `j.MapIndex("foo")`
	//
	// When an error is found, the returned Context is an invalid,
	// and calling methods on it will only return the original error
	MapIndex(string) Context

	stdlib.Marshaler

	Set(interface{}) Context
	SetMapIndex(string, interface{}) Context

	// Slice assigns the value pointed by the Context to the specified
	// destinatio, which must be a pointer to a slice variable
	// compatible with the original slice.
	//
	// For example, if you already know that the JSON array elements
	// are homogenous and only contain strings, you may pass a pointer to `[]string`.
	// If you are not sure, or the JSON array contains heterogenous
	// elements, then use a pointer to `[]interface{}`, or a pointer to
	// an empty `interface{}`
	//
	// If the values cannot be assigned, an error is returned
	Slice(interface{}) error

	// String assigns the value pointed by the Context to the specified
	// destination, which must be a pointer to a variable compatible
	// with string
	// If the underlying value is not a JSON string, then an error is returned
	String(interface{}) error
}

var rdrPool = sync.Pool{
	New: func() interface{} {
		return &bytes.Reader{}
	},
}

func getReader() *bytes.Reader {
	return rdrPool.Get().(*bytes.Reader)
}

func releaseReader(b *bytes.Reader) {
	b.Reset(nil)
	rdrPool.Put(b)
}

var emptyInterfaceType = reflect.TypeOf((*interface{})(nil)).Elem()

func assignIfCompatible(dst, src reflect.Value) error {
	if dst.Kind() == reflect.Ptr {
		dst = dst.Elem()
	}

	dstT := dst.Type()
	srcT := src.Type()

	if !dst.IsValid() {
		return errors.New(`destination variable is not valid`)
	}

	if !dst.CanSet() {
		return errors.New(`destination variable is not assignable`)
	}

	// If it's an empty interface, just assign.
	if dstT == emptyInterfaceType {
		dst.Set(reflect.ValueOf(src.Interface()))
		return nil
	}

	// If it's straight assignable, do that
	if srcT.AssignableTo(dstT) {
		dst.Set(src)
		return nil
	}

	// If it's convertible, assign after conversion
	if srcT.ConvertibleTo(dstT) {
		dst.Set(src.Convert(dstT))
		return nil
	}

	// If we got here, the only other possible assignment is if we
	// have a container whose element types may differ. In that case
	// the kind of dst and src must match
	if dst.Kind() != src.Kind() {
		return fmt.Errorf(`destination variable kind (%s) and source variable kind (%s) do not match`, dstT.Kind(), srcT.Kind())
	}

	// If it's a container that needs conversion... (array/slice or map)
	switch src.Kind() {
	case reflect.Slice, reflect.Array:
		dstElemT := dstT.Elem()
		srcElemT := srcT.Elem()
		if dst.IsNil() {
			// If the type is specified but the value is nil, then initialize
			// it as a slice of the type specified in the dst
			dst.Set(reflect.MakeSlice(dstT, src.Len(), src.Len()))
		} else {
			// Otherwise we should have a slice/array type.
			// If the destination has less capacity than source length, then we bail
			if dst.Cap() < src.Len() {
				return fmt.Errorf(`destination variable does not hold enough capacity (%d) to assign source (%d)`, dst.Cap(), src.Len())
			}

			// We now know we have enough capacity. If the length don't match,
			// make sure the destination slice has the same length as the source
			for dst.Len() < src.Len() {
				dst = reflect.AppendSlice(dst, reflect.Zero(dstElemT))
			}
		}

		// []interface{} is a special case, because we're going to need
		// to get the actual type using Elem()
		if srcElemT.Kind() == reflect.Interface {
			for i := 0; i < src.Len(); i++ {
				if src.Index(i).Elem().Type().AssignableTo(dstElemT) {
					dst.Index(i).Set(src.Index(i).Elem())
				} else if src.Index(i).Elem().Type().ConvertibleTo(dstElemT) {
					dst.Index(i).Set(src.Index(i).Elem().Convert(dstElemT))
				} else {
					return fmt.Errorf(`cannot convert from %T to %T at position %d of slice`, src.Index(i).Elem(), dstElemT, i)
				}
			}
			return nil
		}

		// If dst and src element types match, then we only need to
		// assign directly.
		if srcElemT.AssignableTo(dstElemT) {
			for i := 0; i < src.Len(); i++ {
				dst.Index(i).Set(src.Index(i))
			}
			return nil
		}

		// If conversion is necessary, do that
		if srcElemT.ConvertibleTo(dstElemT) {
			for i := 0; i < src.Len(); i++ {
				dst.Index(i).Set(src.Index(i).Convert(dstElemT))
			}
			return nil
		}

		// Sometime a good old type conversion is all we need

		return errors.New(`ARGH`)
	case reflect.Map:
		dstElemT := dstT.Elem()

		if dst.IsNil() {
			// If the destination is nil, initialize it as a map specified
			// by dst's Type
			dst.Set(reflect.MakeMapWithSize(dstT, len(src.MapKeys())))
		}

		// map[*]interface{} is a special case, because we're going to need
		// to get the actual type using Elem()
		if src.Type().Elem().Kind() == reflect.Interface {
			for _, key := range src.MapKeys() {
				srcv := src.MapIndex(key).Elem() // Elem() to get the value underneath the interface{}
				if srcv.Type().AssignableTo(dstElemT) {
					dst.SetMapIndex(key, srcv)
				} else if srcv.Type().ConvertibleTo(dstElemT) {
					dst.SetMapIndex(key, srcv.Convert(dstElemT))
				} else {
					return fmt.Errorf(`cannot convert from %T to %T from key %#v of map`, srcv, dst.MapIndex(key), key.Interface())
				}
			}
			return nil
		}

		// TODO I'm obviously getting tired of writing code. punting
		for _, key := range src.MapKeys() {
			dst.SetMapIndex(key, src.MapIndex(key))
		}
	}

	return errors.New(`invalid type`)
}

func New(v interface{}) Context {
	return newCtx(v)
}

func Parse(data []byte) (Context, error) {
	var v interface{}

	r := getReader()
	defer releaseReader(r)

	r.Reset(data)
	dec := stdlib.NewDecoder(r)
	dec.UseNumber()

	if err := dec.Decode(&v); err != nil {
		return nil, errors.Wrap(err, `failed to unmarshal JSON`)
	}

	return newCtx(v), nil
}

func (c *ctx) Slice(dst interface{}) error {
	rv := reflect.ValueOf(dst)
	// rv must be a pointer to a slice or array
	switch {
	case rv.Type() == emptyInterfaceType:
		// var s interface{}
	case rv.Kind() == reflect.Ptr || rv.Kind() == reflect.Interface:
		switch rv.Type().Elem().Kind() {
		case reflect.Slice, reflect.Array:
		default:
			return fmt.Errorf(`destination must be a pointer to a slice/array (%T)`, dst)
		}
	default:
		return fmt.Errorf(`destination must be a pointer to a slice/array (%T)`, dst)
	}

	return assignIfCompatible(rv, c.value)
}

func (c *ctx) Map(dst interface{}) error {
	rv := reflect.ValueOf(dst)
	// rv must be a pointer to a map
	switch {
	case rv.Type() == emptyInterfaceType:
		// var m interface{}
	case rv.Kind() == reflect.Ptr || rv.Kind() == reflect.Interface:
		// var m map[...]...
		if rv.Type().Elem().Kind() != reflect.Map {
			return fmt.Errorf(`destination must be a pointer to a map (%T)`, dst)
		}
		// We also only support string keys
		if rv.Type().Elem().Key().Kind() != reflect.String {
			return fmt.Errorf(`destination map must use a string key`)
		}
	default:
		return fmt.Errorf(`destination must be a pointer to a map (%T)`, dst)
	}

	return assignIfCompatible(rv, c.value)
}

func (c *ctx) Bool(dst interface{}) error {
	rv := reflect.ValueOf(dst)

	switch {
	case rv.Type() == emptyInterfaceType:
	case rv.Kind() == reflect.Ptr || rv.Kind() == reflect.Interface:
		if rv.Type().Elem().Kind() != reflect.Bool {
			return fmt.Errorf(`destination must be a pointer to bool (%T)`, dst)
		}
	default:
		return fmt.Errorf(`destination must be a pointer to bool (%T)`, dst)
	}

	return assignIfCompatible(rv, c.value)
}

func (c *ctx) Float(dst interface{}) error {
	rv := reflect.ValueOf(dst)

	switch {
	case rv.Type() == emptyInterfaceType:
	case rv.Kind() == reflect.Ptr || rv.Kind() == reflect.Interface:
		switch rv.Type().Elem().Kind() {
		case reflect.Float32, reflect.Float64:
		default:
			return fmt.Errorf(`destination must be a pointer to float32/float64 (%T)`, dst)
		}
	default:
		return fmt.Errorf(`destination must be a pointer to float32/float64 (%T)`, dst)
	}

	n, ok := c.value.Interface().(stdlib.Number)
	if !ok {
		return fmt.Errorf(`failed to assert %T into a json.Number type`, c.value.Interface())
	}

	f, err := n.Float64()
	if err != nil {
		return fmt.Errorf(`failed to convert json.Number into float64: %s`, err)
	}

	return assignIfCompatible(rv, reflect.ValueOf(f))
}

func (c *ctx) Int(dst interface{}) error {
	rv := reflect.ValueOf(dst)

	switch {
	case rv.Type() == emptyInterfaceType:
	case rv.Kind() == reflect.Ptr || rv.Kind() == reflect.Interface:
		switch rv.Type().Elem().Kind() {
		case reflect.Int, reflect.Int8, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint32, reflect.Uint64:
		default:
			return fmt.Errorf(`destination must be a pointer to int/int8/int32/int64/uint/uint8/uint16/uint32/uint64 (%T)`, dst)
		}
	default:
		return fmt.Errorf(`destination must be a pointer to int/int8/int32/int64/uint/uint8/uint16/uint32/uint64 (%T)`, dst)
	}

	n, ok := c.value.Interface().(stdlib.Number)
	if !ok {
		return fmt.Errorf(`failed to assert %T into a json.Number type`, c.value.Interface())
	}

	i, err := n.Int64()
	if err != nil {
		return fmt.Errorf(`failed to convert json.Number into int: %s`, err)
	}

	return assignIfCompatible(rv, reflect.ValueOf(i))
}

func (c *ctx) String(dst interface{}) error {
	rv := reflect.ValueOf(dst)

	switch {
	case rv.Type() == emptyInterfaceType:
	case rv.Kind() == reflect.Ptr || rv.Kind() == reflect.Interface:
		if rv.Type().Elem().Kind() != reflect.String {
			return fmt.Errorf(`destination must be a pointer to string (%T)`, dst)
		}
	default:
		return fmt.Errorf(`destination must be a pointer to string (%T)`, dst)
	}

	return assignIfCompatible(rv, c.value)
}

func (c *ctx) MapIndex(n string) Context {
	if c.value.Kind() != reflect.Map {
		return newErrCtx(fmt.Errorf(`cannot access field %#v of non-map type (%T)`, n, c.value.Interface()))
	}

	keyV := reflect.ValueOf(n)
	v := c.value.MapIndex(keyV)
	if v == zeroval {
		return newErrCtx(fmt.Errorf(`field %#v not found`, n))
	}

	c2 := newCtx(v.Interface())

	parent := c.value
	c2.set = func(v reflect.Value) {
		parent.SetMapIndex(keyV, v)
	}
	return c2
}

func (c *ctx) Index(i int) Context {
	switch c.value.Kind() {
	case reflect.Slice, reflect.Array:
	default:
		return newErrCtx(fmt.Errorf(`cannot access index %d of non-slice/array type (%T)`, i, c.value.Interface()))
	}

	if i < 0 || c.value.Len() <= i {
		// note: this particular error needs no stack, using fmt
		return newErrCtx(fmt.Errorf(`index %d is out of bounds (len=%d)`, i, c.value.Len()))
	}

	v := c.value.Index(i)
	c2 := newCtx(v.Interface())

	parent := c.value
	c2.set = func(v reflect.Value) {
		parent.Index(i).Set(v)
	}
	return c2
}

func (c *ctx) Set(v interface{}) Context {
	if c.value == zeroval {
		c.value = reflect.ValueOf(v)
	} else {
		if set := c.set; set != nil {
			set(reflect.ValueOf(v))
		} else {
			if !c.value.CanSet() {
				panic(fmt.Sprintf("%#v", c.value.Interface()))
			}
			c.value.Set(reflect.ValueOf(v))
		}
	}
	return c
}

func (c *ctx) SetMapIndex(key string, value interface{}) Context {
	if c.value.Kind() != reflect.Map {
		return newErrCtx(fmt.Errorf(`cannot set field %#v of non-map type (%T)`, key, c.value.Interface()))
	}

	c.value.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(value))

	return c
}

func (c *ctx) MarshalJSON() ([]byte, error) {
	return stdlib.Marshal(c.value.Interface())
}
