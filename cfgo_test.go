package cfgo

import (
	"fmt"
	"testing"
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

func Test1(t *testing.T) {
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
	fmt.Printf("strucePtr1: %+v\n", strucePtr1)
	fmt.Printf("strucePtr2: %+v\n", strucePtr2)
	fmt.Printf("strucePtr3: %+v\n", strucePtr3)
}
