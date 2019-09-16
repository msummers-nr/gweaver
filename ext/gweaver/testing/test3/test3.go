// +weaver packageFQN  gweaver/testing/test3
package test3

// +weaver insert
import "fmt"

// // +weaver delete
var deleteVar string // +weaver delete

// +weaver replace
var replaceMeWithString string

// +weaver insert
var insertedVar string

// // +weaver delete
type deleteStruct struct // +weaver delete
{

}

// +weaver replace
type replaceStruct struct {
	fieldAddedToReplaceStruct string
}

// +weaver insert
type structInserted struct{}

type structWithFieldToDelete struct {
	deleteMe string // +weaver delete
}

type structWithFieldToInsert struct {
	one   string
	two   string // +weaver insert
	three string
}

type structWithFieldToReplace struct {
	dontReplace      string
	replaceMeWithInt int // +weaver replace
}

type obj struct{}

func Main() {
}

// +weaver delete
func funcToDelete() {
	// Delete me!
}

// +weaver replaceAndCallOriginal
func funcToUpdate() {
	// Mumble
	fmt.Println("I'm updated!")
	funcToUpdateOriginal()
	// Mumble
}

// +weaver insert
func funcInserted() {
}

// +weaver delete
func (o *obj) methodToDelete() {

}

// +weaver replaceAndCallOriginal
func (o *obj) methodToUpdate() {
	// Mumble
	fmt.Println("I'm updated!")
	o.methodToUpdateOriginal()
	// Mumble
}

// +weaver insert
func (o *obj) methodInserted() {
}
