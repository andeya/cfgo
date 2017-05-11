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
	fmt.Printf("strucePtr1(config.yaml): %+v\n\n", strucePtr1)
}
