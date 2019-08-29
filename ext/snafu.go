// // +build weaver
// //go:generate weaver

package gweaver

import "fmt"

//
// redefine throws an error if it can't find the original
//

// +weaving replace
type obj struct {
	msg string
}

// +weaving surround
func func1() {
	fmt.Println("Before func1")
	// This looks like recursion but it's really a call to the ORIGINAL func1. Should the syntax be different?
	func1()
	fmt.Println("After func1")
}

// +weaving replace
func (o *obj) method1() {
	fmt.Println("Before method1 " + o.msg)
	// This looks like recursion but it's really a call to the ORIGINAL method1. Should the syntax be different?
	o.method1()
	fmt.Println("After method1 " + o.msg)
}

//
// new throws an error if there is a collision with the existing code
//

// +weaving create
type snafu struct{}

// +weaving create
var fubar string

// +weaving create
func someFunction() {}

// +weaving create
func (o *obj) someMethod() {}
