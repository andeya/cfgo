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
