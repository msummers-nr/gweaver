package snafu

import "fmt"

type obj struct{}

func Main() {
	func1()
	o := &obj{}
	o.method1()
}

func func1() {
	fmt.Println("In func1")
}

func (o *obj) method1() {
	fmt.Println("In method1")
}
