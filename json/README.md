# SYNOPSIS

WARNING: As of this writing (Jun 2020), this is a PoC

## Parse and extract values from a JSON document

```go
import (
	"fmt"

  "github.com/lestrrat-go/json"
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

  j, err := json.Parse(src)
  if err != nil {
    return fmt.Errorf(`failed to parse JSON: %w`, err)
  }

  var s1 string
  if err := j.MapIndex("foo").String(&s1); err != nil {
    fmt.Printf("failed to get value of 'foo': %s\n", err)
  	return
  }
}
```

## Build a structure to be serialized

```
import (
	"fmt"
	stdlib "encoding/json"

  "github.com/lestrrat-go/json"
)

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
}
```
