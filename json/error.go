package json

func (c errCtx) Bool(_ interface{}) error {
	return c.err
}

func (c errCtx) Float(_ interface{}) error {
	return c.err
}

func (c errCtx) Index(_ int) Context {
	return c
}

func (c errCtx) Int(_ interface{}) error {
	return c.err
}

func (c errCtx) Map(_ interface{}) error {
	return c.err
}

func (c errCtx) MapIndex(_ string) Context {
	return c
}

func (c errCtx) Set(_ interface{}) Context {
	return c
}

func (c errCtx) SetMapIndex(_ string, _ interface{}) Context {
	return c
}

func (c errCtx) Slice(_ interface{}) error {
	return c.err
}

func (c errCtx) String(_ interface{}) error {
	return c.err
}

func (c errCtx) MarshalJSON() ([]byte, error) {
	return nil, c.err
}
