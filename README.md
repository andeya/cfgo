# cfgo  [![GoDoc](https://godoc.org/github.com/tsuna/gohbase?status.png)](https://godoc.org/github.com/henrylee2cn/cfgo)

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

func (t *T1) Reload(bind cfgo.BindFunc) error {
	fmt.Println("module_1: T1 reload do some thing...")
	return bind()
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
	fmt.Printf("strucePtr1(config/config.yaml): %+v\n\n", strucePtr1)
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

func (t *T2) Reload(bind cfgo.BindFunc) error {
	fmt.Println("module_2: T2 reload do some thing...")
	return bind()
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
	fmt.Printf("strucePtr2(config/config.yaml): %+v\n\n", strucePtr2)
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

func (t *T) Reload(bind cfgo.BindFunc) error {
	fmt.Println("main T reload do some thing...")
	return bind()
}

func main() {
	strucePtr := &T{
		C: "c",
		T1: m1.T1{
			B: 2, //default value
		},
	}

	// output: config/config.yaml
	c := cfgo.MustGet("config/config.yaml")
	c.MustReg("section", strucePtr)
	// or
	// cfgo.MustReg("section", strucePtr)

	fmt.Printf("strucePtr(config/config.yaml): %+v\n\n", strucePtr)

	// output: config/config3.yaml
	c3 := cfgo.MustGet("config/config3.yaml")
	c3.MustReg("section", strucePtr)
	fmt.Printf("strucePtr(config/config3.yaml): %+v\n\n", strucePtr)

	fmt.Printf(" ----------------------------------------------------------- \n\n")

	fmt.Printf("config/config.yaml content:\n%s\n\n", c.Content())
	// or
	// fmt.Printf("config.yaml content:\n%s\n\n", cfgo.Content())

	fmt.Printf(" ----------------------------------------------------------- \n\n")

	fmt.Printf("config/config3.yaml content:\n%s\n\n", c3.Content())
}
```

print:

```
module_1: T1 reload do some thing...
strucePtr1(config/config.yaml): &{F:1 B:2}

module_2: T2 reload do some thing...
strucePtr2(config/config.yaml): &{X:xxx Y:[x y z] Z:[1 2 3] N:false}

=== RUN   Test1
main T reload do some thing...
strucePtr(config/config.yaml): &{C:c T1:{F:0 B:2}}

main T reload do some thing...
strucePtr(config/config3.yaml): &{C:c T1:{F:0 B:2}}

 ----------------------------------------------------------- 

config/config.yaml content:
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


 ----------------------------------------------------------- 

config/config3.yaml content:
section:
  c: c
  t1:
    b: 2

# ------------------------- non-automated configuration -------------------------

custom:
  true

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

# custom

Cfgo allows to custom non-automated configuration in yaml file.

example `config/mixed_config.yaml`:

```
# cfgo-register section
register:
  auto: true

# your custom section
custom: true

```

golang code main.go:

```
package main

import (
	"fmt"

	"github.com/henrylee2cn/cfgo"
)

type M struct {
	Auto bool
}

func (m *M) Reload(bind cfgo.BindFunc) error {
	return bind()
}

func main() {
	m := new(M)
	mixed := cfgo.MustGet("config/mixed_config.yaml")
	mixed.MustReg("register", m)

	fmt.Printf("config/mixed_config.yaml content:\n%s\n\n", mixed.Content())
	fmt.Printf("config/mixed_config.yaml config m:\n%#v\n\n", m)
	{
		custom, _ := mixed.GetConfig("custom")
		fmt.Printf("config/mixed_config.yaml GetConfig 'custom':\n%#v\n\n", custom)
	}
	{
		var custom bool
		_ = mixed.BindConfig("custom", &custom)
		fmt.Printf("config/mixed_config.yaml BindConfig 'custom':\n%#v\n\n", custom)
	}
}
```

Run the command `go run main.go`, will print:

```
config/mixed_config.yaml content:
register:
  auto: true

# ------------------------- non-automated configuration -------------------------

custom:
  true


config/mixed_config.yaml config m:
&main.M{Auto:true}

config/mixed_config.yaml GetConfig 'custom':
true

config/mixed_config.yaml BindConfig 'custom':
true
```

now `config/mixed_config.yaml` content auto-change to:

```
register:
  auto: true

# ------------------------- non-automated configuration -------------------------

custom:
  true

```
