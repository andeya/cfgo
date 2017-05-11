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

    (...) yaml:"[<key>][,<flag1>[,<flag2>]]" (...)

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

In addition, if the key is `-`, the field is ignored.

# example

- m1/module_1.go

```go
package m1

import (
	"fmt"

	"github.com/henrylee2cn/cfgo"
)

type T1 struct {
	F int `yaml:"a,omitempty"`
	B int
}

func (t T1) Reload() error {
	fmt.Println("module_1: T1 reload do some thing...")
	return nil
}

func init() {
	strucePtr1 := &T1{
		F: 1, //default value
		B: 2, //default value
	}
	{
		c := cfgo.MustGet("config/config.yaml")
		c.MustReg("section1", strucePtr1)
	}
	// or
	// cfgo.MustReg("section1", strucePtr1)
	fmt.Printf("strucePtr1(config.yaml): %+v\n\n", strucePtr1)
}

```

- m2/module_2.go

```go
package m2

import (
	"fmt"

	"github.com/henrylee2cn/cfgo"
)

type T2 struct {
	X string
	Y []string `yaml:",flow"`
	Z []int
	N bool
}

func (t T2) Reload() error {
	fmt.Println("module_2: T2 reload do some thing...")
	return nil
}

func init() {
	strucePtr2 := &T2{
		X: "xxx",                   //default value
		Y: []string{"x", "y", "z"}, //default value
		Z: []int{1, 2, 3},          //default value
	}
	{
		c := cfgo.MustGet("config/config.yaml")
		c.MustReg("section2", strucePtr2)
	}
	// or
	// cfgo.MustReg("section2", strucePtr2)
	fmt.Printf("strucePtr2(config.yaml): %+v\n\n", strucePtr2)
}

```

- main.go

```go
package main

import (
	"fmt"

	"github.com/henrylee2cn/cfgo"
	"github.com/henrylee2cn/cfgo/example/m1"
	_ "github.com/henrylee2cn/cfgo/example/m2"
)

type T struct {
	C string
	m1.T1
}

func (t T) Reload() error {
	fmt.Println("T reload do some thing...")
	return nil
}

func main() {
	strucePtr := &T{
		C: "c",
		T1: m1.T1{
			B: 2, //default value
		},
	}

	// output: config/config3.yaml
	{
		c := cfgo.MustGet("config/config3.yaml")
		c.MustReg("section", strucePtr)
		fmt.Printf("strucePtr(config3.yaml): %+v\n\n", strucePtr)
	}

	// output: config/config.yaml
	{
		c := cfgo.MustGet("config/config.yaml")
		c.MustReg("section", strucePtr)
	}
	// or
	// cfgo.MustReg("section", strucePtr)
	fmt.Printf("strucePtr(config.yaml): %+v\n\n", strucePtr)
}

```

print:

```
module_1: T1 reload do some thing...
strucePtr1(config.yaml): &{F:1 B:2}

module_2: T2 reload do some thing...
strucePtr2(config.yaml): &{X:xxx Y:[x y z] Z:[1 2 3] N:false}

T reload do some thing...
strucePtr(config3.yaml): &{C:c T1:{F:0 B:2}}

T reload do some thing...
strucePtr(config.yaml): &{C:c T1:{F:0 B:2}}
```

output `config/config.yaml`:

```
section:
  c: c
  t1:
    b: 2

section1:
  a: 1
  b: 2

section2:
  x: xxx
  "y": [x, "y", z]
  z:
  - 1
  - 2
  - 3
  "n": false

```

output `config/config3.yaml`:

```
section:
  c: c
  t1:
    b: 2

```
