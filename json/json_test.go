package json_test

import (
	stdlib "encoding/json"
	"fmt"
	"testing"

	"github.com/lestrrat-go/json"
	"github.com/stretchr/testify/assert"
)

func ExampleParse() {
	const src = `{
  "foo": "bar",
  "number_int": 1,
  "number_float": 1.234,
  "bool": true,
  "array": [ "one", 2, 3.0 ],
  "map": {
    "sub1": 1,
    "sub2": 2,
    "sub3": 3
  }
}`

	j, err := json.Parse([]byte(src))
	if err != nil {
		fmt.Printf("failed to parse JSON: %s\n", err)
		return
	}

	var s1 string
	if err := j.MapIndex("foo").String(&s1); err != nil {
		fmt.Printf("failed to get value of 'foo': %s\n", err)
		return
	}

	//OUTPUT:
}

func ExampleBuild() {
	j := json.New(map[string]interface{}{}).
		SetMapIndex("foo", "bar").
		SetMapIndex("number_int", 1).
		SetMapIndex("number_float", 1.234).
		SetMapIndex("bool", true).
		SetMapIndex("array", []interface{}{"one", 2, 3.0}).
		SetMapIndex("map", map[string]interface{}{"sub1": 1, "sub2": 2, "sub3": 3})

	buf, err := stdlib.Marshal(j)
	if err != nil {
		fmt.Printf("failed to marshal json: %s", err)
		return
	}
	fmt.Printf("%s\n", buf)
	//OUTPUT:
	//{"array":["one",2,3],"bool":true,"foo":"bar","map":{"sub1":1,"sub2":2,"sub3":3},"number_float":1.234,"number_int":1}
}

func Benchmark(b *testing.B) {
	b.Run("Extract a string slice", func(b *testing.B) {
		const mapsrc = `["hello", "world", "foo", "bar", "baz"]`
		j, err := json.Parse([]byte(mapsrc))
		if err != nil {
			b.Errorf(`json.Parse failed: %s`, err)
			return
		}

		var s []string
		for i := 0; i < b.N; i++ {
			if err := j.Slice(&s); err != nil {
				b.Errorf(`j.Slice failed: %s`, err)
				return
			}
			_ = s
		}
	})
	b.Run("Extract a string map", func(b *testing.B) {
		const mapsrc = `{"hello": "world", "foo": "bar", "baz": "quux"}`
		j, err := json.Parse([]byte(mapsrc))
		if err != nil {
			b.Errorf(`json.Parse failed: %s`, err)
			return
		}

		var m map[string]string
		for i := 0; i < b.N; i++ {
			if err := j.Map(&m); err != nil {
				b.Errorf(`j.Map failed: %s`, err)
				return
			}
			_ = m
		}
	})
	b.Run("Extract a complex map", func(b *testing.B) {
		const mapsrc = `{"hello": "world", "foo": 1, "bar": null, "baz": 1.234, "nested": { "hello": "nested world" } }`
		j, err := json.Parse([]byte(mapsrc))
		if err != nil {
			b.Errorf(`json.Parse failed: %s`, err)
			return
		}

		var m map[string]interface{}
		for i := 0; i < b.N; i++ {
			if err := j.Map(&m); err != nil {
				b.Errorf(`j.Map failed: %s`, err)
				return
			}
			_ = m
		}
	})
}

func TestMap(t *testing.T) {
	t.Run("sanity", func(t *testing.T) {
		const mapsrc = `{"hello": "world", "foo": 1, "bar": null, "baz": 1.234, "nested": { "hello": "nested world" } }`
		j, err := json.Parse([]byte(mapsrc))
		if !assert.NoError(t, err, `json.Parse should succeed`) {
			return
		}

		var m map[string]interface{}
		if !assert.NoError(t, j.Map(&m), `j.Map should succeed1`) {
			return
		}

		if !assert.Equal(t, "world", m["hello"], `values should match`) {
			return
		}

		var s1 string
		if !assert.NoError(t, j.MapIndex("nested").MapIndex("hello").String(&s1), `j.MapIndex.MapIndex.String should succeed`) {
			return
		}
		if !assert.Equal(t, "nested world", s1, `string values should match`) {
			return
		}

		if !assert.Error(t, j.MapIndex("nested").MapIndex("non-exixstent").String(&s1), `j.MapIndex.MapIndex.String should fail`) {
			return
		}

		var i1 int
		if !assert.NoError(t, j.MapIndex("foo").Int(&i1), `j.MapIndex.Int should succeed`) {
			return
		}
		if !assert.Equal(t, 1, i1, `values should match`) {
			return
		}

		var f1 float64
		if !assert.NoError(t, j.MapIndex("baz").Float(&f1), `j.MapIndex.Float should succeed`) {
			return
		}
		if !assert.Equal(t, 1.234, f1, `values should match`) {
			return
		}
	})
	t.Run("assign to a map[string]string", func(t *testing.T) {
		const mapsrc = `{"foo": "bar", "baz": "quux", "hoge": "fuga" }`
		j, err := json.Parse([]byte(mapsrc))
		if !assert.NoError(t, err, `json.Parse should succeed`) {
			return
		}

		var m map[string]string
		if !assert.NoError(t, j.Map(&m), `j.Map should succeed`) {
			return
		}
		if !assert.Equal(t, "bar", m["foo"], `values should match (1)`) {
			return
		}
	})

	t.Run("invalid data for Map", func(t *testing.T) {
		const arraysrc = `["hello", 1, true, null, 1.234]`

		tests := []struct {
			Name string
			Src  string
		}{
			{Name: "array", Src: arraysrc},
		}

		for _, data := range tests {
			data := data
			t.Run(data.Name, func(t *testing.T) {
				j, err := json.Parse([]byte(data.Src))
				if !assert.NoError(t, err, `json.Parse should succeed`) {
					return
				}
				var m map[string]interface{}
				if !assert.Error(t, j.Map(&m), `j.Map should fail`) {
					return
				}
			})
		}
	})
}

func TestArray(t *testing.T) {
	t.Run("sanity", func(t *testing.T) {
		const arraysrc = `["hello", 1.2345, true, null, {} ]`
		j, err := json.Parse([]byte(arraysrc))
		if !assert.NoError(t, err, `json.Parse should succeed`) {
			return
		}

		var dst []interface{}
		if err := j.Slice(&dst); !assert.NoError(t, err, `j.Slice should succeed`) {
			return
		}

		s1, ok := dst[0].(string)
		if !assert.True(t, ok, `dst[0] should be a string`) {
			return
		}

		var s2 string
		if !assert.NoError(t, j.Index(0).String(&s2), `j.Index should succeed`) {
			return
		}

		if !assert.Equal(t, "hello", s1, `values should match (1)`) {
			return
		}
		if !assert.Equal(t, s1, s2, `values should match (2)`) {
			return
		}
	})
	t.Run("assigning to []string", func(t *testing.T) {
		const arraysrc = `["hello", "world", "woohoo"]`
		j, err := json.Parse([]byte(arraysrc))
		if !assert.NoError(t, err, `json.Parse should succeed`) {
			return
		}

		var dst []string
		if err := j.Slice(&dst); !assert.NoError(t, err, `j.Slice should succeed`) {
			return
		}

		if !assert.Equal(t, []string{"hello", "world", "woohoo"}, dst, `values should match`) {
			return
		}
	})
}

func TestBuild(t *testing.T) {
	t.Run("Set a single value", func(t *testing.T) {
		j := json.New(nil).Set(1)
		buf, err := j.MarshalJSON()
		if !assert.NoError(t, err, `j.MarshalJSON should succeed`) {
			return
		}

		if !assert.Equal(t, "1", string(buf), `json string should match`) {
			return
		}
	})
	t.Run("Set a value within a slice", func(t *testing.T) {
		j := json.New([]string{"hello", "foo", "bar", "baz"})

		j.Index(2).Set("hoge")

		buf, err := j.MarshalJSON()
		if !assert.NoError(t, err, `j.MarshalJSON should succeed`) {
			return
		}

		if !assert.Equal(t, `["hello","foo","hoge","baz"]`, string(buf), `json string should match`) {
			return
		}
	})
	t.Run("Set a value within a map", func(t *testing.T) {
		j := json.New(map[string]interface{}{"hello": "world"})

		j.MapIndex("hello").Set("neighbor")

		buf, err := j.MarshalJSON()
		if !assert.NoError(t, err, `j.MarshalJSON should succeed`) {
			return
		}

		if !assert.Equal(t, `{"hello":"neighbor"}`, string(buf), `json string should match`) {
			return
		}
	})
}
