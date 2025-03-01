package parts

import "testing"

func TestHelperNoTags(t *testing.T) {
	type TestStruct struct {
		Name string
		Age  int
	}

	vm, err := GetVMWithSource("let Name = \"tfo\"; let Age = 22")

	if err != nil {
		t.Error(err)
		return
	}

	vm.Run()

	var testStruct TestStruct

	ReadFromParts(vm, &testStruct)

	if testStruct.Age != 22 && testStruct.Name != "tfo" {
		t.Errorf("inexpected values in the interface got (%d, %s) expected (22, \"tfo\")", testStruct.Age, testStruct.Name)
		return
	}
}

func TestHelperWithTags(t *testing.T) {
	type TestStruct struct {
		Name string `parts:"name"`
		Age  int `parts:"age"`
	}

	vm, err := GetVMWithSource("let name = \"tfo\"; let age = 22")

	if err != nil {
		t.Error(err)
		return
	}

	vm.Run()

	var testStruct TestStruct

	ReadFromParts(vm, &testStruct)

	if testStruct.Age != 22 && testStruct.Name != "tfo" {
		t.Errorf("inexpected values in the interface got (%d, %s) expected (22, \"tfo\")", testStruct.Age, testStruct.Name)
		return
	}
}