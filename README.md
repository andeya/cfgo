# cfgo
Cfgo from the YAML document, bi-directional synchronous multi-module configuration.

The structure of the generated document will reflect the structure of the value itself.

Maps and pointers (to struct, string, int, etc) are accepted as the in value.

Struct fields are only unmarshalled if they are exported (have an upper case
first letter), and are unmarshalled using the field name lowercased as the
default key. Custom keys may be defined via the "yaml" name in the field
tag: the content preceding the first comma is used as the key, and the
following comma-separated options are used to tweak the marshalling process.
Conflicting names result in a runtime error.

The field tag format accepted is:

    `(...) yaml:"[<key>][,<flag1>[,<flag2>]]" (...)`

The following flags are currently supported:

    omitempty    Only include the field if it's not set to the zero
                 value for the type or to empty slices or maps.
                 Does not apply to zero valued structs.

    flow         Marshal using a flow style (useful for structs,
                 sequences and maps).

    inline       Inline the field, which must be a struct or a map,
                 causing all of its fields or keys to be processed as if
                 they were part of the outer struct. For maps, keys must
                 not conflict with the yaml keys of other struct fields.

In addition, if the key is "-", the field is ignored.

# example

```go
package main

import (
	"fmt"
)

type T struct {
	F int `yaml:"a,omitempty"`
	B int
}

func (t T) Reload() error {
	fmt.Println("T reload do some thing...")
	return nil
}

type T2 struct {
	X string
	Y []string `yaml:",flow"`
	Z []int
	N bool
}

func (t T2) Reload() error {
	fmt.Println("T2 reload do some thing...")
	return nil
}

func main() {
	strucePtr1 := &T{
		B: 2, //default value
	}
	strucePtr2 := &T{
		F: 11, //default value
		B: 22, //default value
	}
	strucePtr3 := &T2{
		X: "xxx",                   //default value
		Y: []string{"x", "y", "z"}, //default value
		Z: []int{1, 2, 3},          //default value
	}
	MustReg("section1", strucePtr1)
	MustReg("section2", strucePtr2)
	MustReg("section3", strucePtr3)
	fmt.Printf("strucePtr1: %+v", strucePtr1)
	fmt.Printf("strucePtr2: %+v", strucePtr2)
	fmt.Printf("strucePtr3: %+v", strucePtr3)
}
```

output `config/config.yaml`:

```
section1:
  b: 2

section2:
  a: 11
  b: 22

section3:
  x: xxx
  "y": [x, "y", z]
  z:
  - 1
  - 2
  - 3
  "n": false

```