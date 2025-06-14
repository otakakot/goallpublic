package sample

import "fmt"

var _ sampleInterface = (*sampleStruct)(nil)

var sampleVar = "This is a sample variable."

const sampleConst = "This is a sample constant."

type sampleInterface interface {
	sampleMethod()
}

type sampleStruct struct {
	sampleField string
}

func (ss sampleStruct) sampleMethod() {
	fmt.Println(ss.sampleField)
}

func sampleFunc() {
	sample := sampleStruct{sampleField: "example"}
	fmt.Printf("Sample Field: %s\n", sample.sampleField)
	sample.sampleMethod()
}

func main() {
	sampleFunc()
}
