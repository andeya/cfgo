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
	fmt.Printf("strucePtr2(config.yaml): %+v\n\n", strucePtr2)
}
