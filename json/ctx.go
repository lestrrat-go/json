package json

import "reflect"

// a errCtx exists solely to propagate an error that occurred
// during a chained execution.
type errCtx struct {
	err error
}

type ctx struct {
	set func(reflect.Value)
	value reflect.Value
}

func newCtx(v interface{}) *ctx {
	return &ctx{value: reflect.ValueOf(v)}
}

func newErrCtx(e error) *errCtx {
	return &errCtx{err: e}
}
