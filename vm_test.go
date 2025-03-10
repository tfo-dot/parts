package parts

import (
	"errors"
	"testing"
)

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

	err = vm.Run()

	if err != nil {
		t.Error(err)
		return
	}

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
		Age  int    `parts:"age"`
	}

	vm, err := GetVMWithSource("let name = \"tfo\"; let age = 22")

	if err != nil {
		t.Error(err)
		return
	}

	err = vm.Run()

	if err != nil {
		t.Error(err)
		return
	}

	var testStruct TestStruct

	ReadFromParts(vm, &testStruct)

	if testStruct.Age != 22 && testStruct.Name != "tfo" {
		t.Errorf("inexpected values in the interface got (%d, %s) expected (22, \"tfo\")", testStruct.Age, testStruct.Name)
		return
	}
}

func TestHelperWithList(t *testing.T) {
	type TestStruct struct {
		Flags []int `parts:"flags"`
	}

	vm, err := GetVMWithSource("let flags = [1, 2, 4, 8]")

	if err != nil {
		t.Error(err)
		return
	}

	err = vm.Run()

	if err != nil {
		t.Error(err)
		return
	}

	var testStruct TestStruct

	ReadFromParts(vm, &testStruct)

	expectedValue := []int{1, 2, 4, 8}

	if len(testStruct.Flags) != len(expectedValue) {
		t.Errorf("array length didn't matched got (%d) expected (%d)", len(testStruct.Flags), len(expectedValue))
		return
	}

	for i, val := range testStruct.Flags {
		if val != expectedValue[i] {
			t.Errorf("inexpected values in the interface got (%d) expected (%d)", val, expectedValue[i])
			return
		}
	}
}

func TestHelperWithObject(t *testing.T) {
	type TestStruct struct {
		Loot struct {
			Exp  int `parts:"exp"`
			Gold int `parts:"gold"`
		} `parts:"loot"`
	}

	vm, err := GetVMWithSource("let loot = |> exp: 100, gold: 200 <|")

	if err != nil {
		t.Error(err)
		return
	}

	err = vm.Run()

	if err != nil {
		t.Error(err)
		return
	}

	var testStruct TestStruct

	ReadFromParts(vm, &testStruct)

	if testStruct.Loot.Exp != 100 {
		t.Errorf("field value didn't matched got (%d) expected (%d)", testStruct.Loot.Exp, 100)
		return
	}

	if testStruct.Loot.Gold != 200 {
		t.Errorf("field value didn't matched got (%d) expected (%d)", testStruct.Loot.Gold, 200)
		return
	}
}

func TestHelperJoined(t *testing.T) {
	type LootStruct struct {
		Type  int
		Count int
	}

	type TestStruct struct {
		Id   string
		HP   int
		SPD  int
		ATK  int
		Name string

		Loot []LootStruct
	}

	vm, err := GetVMWithSource(`
		let Id = "LV0_Dragon"
		let HP = 300
		let SPD = 40
		let ATK = 60
		let Name = "Smok"

		let Loot = [
		  |> Type: 1,  Count: 130 <|,
		  |> Type: 2, Count: 315 <|
		]`)

	if err != nil {
		t.Error(err)
		return
	}

	err = vm.Run()

	if err != nil {
		t.Error(err)
		return
	}

	var testStruct TestStruct

	ReadFromParts(vm, &testStruct)

	if testStruct.Id != "LV0_Dragon" {
		t.Errorf("field value didn't matched got (%s) expected (%s)", testStruct.Id, "LV0_Dragon")
		return
	}

	if testStruct.HP != 300 {
		t.Errorf("field value didn't matched got (%d) expected (%d)", testStruct.HP, 300)
		return
	}

	if testStruct.SPD != 40 {
		t.Errorf("field value didn't matched got (%d) expected (%d)", testStruct.SPD, 40)
		return
	}

	if testStruct.ATK != 60 {
		t.Errorf("field value didn't matched got (%d) expected (%d)", testStruct.ATK, 60)
		return
	}

	if testStruct.Name != "Smok" {
		t.Errorf("field value didn't matched got (%s) expected (%s)", testStruct.Name, "Smok")
		return
	}

	expectedLoot := []LootStruct{{Type: 1, Count: 130}, {Type: 2, Count: 315}}

	for i, val := range testStruct.Loot {
		if val.Count != expectedLoot[i].Count {
			t.Errorf("inexpected values in the interface got (%d) expected (%d)", val.Count, expectedLoot[i].Count)
			return
		}

		if val.Type != expectedLoot[i].Type {
			t.Errorf("inexpected values in the interface got (%d) expected (%d)", val.Type, expectedLoot[i].Type)
			return
		}
	}
}

func TestDotObj(t *testing.T) {
	type TestStruct struct {
		Val int
	}

	vm, err := GetVMWithSource(`
		let obj = |> key: 123 <|
		let Val = obj.key `)

	if err != nil {
		t.Error(err)
		return
	}

	err = vm.Run()

	if err != nil {
		t.Error(err)
		return
	}

	var testStruct TestStruct

	ReadFromParts(vm, &testStruct)

	if testStruct.Val != 123 {
		t.Errorf("field value didn't matched got (%d) expected (%d)", testStruct.Val, 123)
		return
	}

}

func TestDotList(t *testing.T) {
	type TestStruct struct {
		Val int
	}

	vm, err := GetVMWithSource(`
		let obj = [10, 20]
		let Val = obj.0
	`)

	if err != nil {
		t.Error(err)
		return
	}

	err = vm.Run()

	if err != nil {
		t.Error(err)
		return
	}

	var testStruct TestStruct

	ReadFromParts(vm, &testStruct)

	if testStruct.Val != 10 {
		t.Errorf("field value didn't matched got (%d) expected (%d)", testStruct.Val, 10)
		return
	}
}

func TestDotListRef(t *testing.T) {
	type TestStruct struct {
		Val int
	}

	vm, err := GetVMWithSource(`let idx = 0
		let obj = [10]
		let Val = obj[idx]`)

	if err != nil {
		t.Error(err)
		return
	}

	err = vm.Run()

	if err != nil {
		t.Error(err)
		return
	}

	var testStruct TestStruct

	ReadFromParts(vm, &testStruct)

	if testStruct.Val != 10 {
		t.Errorf("field value didn't matched got (%d) expected (%d)", testStruct.Val, 10)
		return
	}
}

func TestDotListAccess(t *testing.T) {
	type TestStruct struct {
		Val int
	}

	vm, err := GetVMWithSource(`let obj = [10, 20]
		let Val = obj[0]`)

	if err != nil {
		t.Error(err)
		return
	}

	err = vm.Run()

	if err != nil {
		t.Error(err)
		return
	}

	var testStruct TestStruct

	ReadFromParts(vm, &testStruct)

	if testStruct.Val != 10 {
		t.Errorf("field value didn't matched got (%d) expected (%d)", testStruct.Val, 10)
		return
	}
}

func TestVarAssign(t *testing.T) {
	type TestStruct struct {
		Val int `parts:"num"`
	}

	vm, err := GetVMWithSource(`let num = 10
		num = 15`)

	if err != nil {
		t.Error(err)
		return
	}

	err = vm.Run()

	if err != nil {
		t.Error(err)
		return
	}

	var testStruct TestStruct

	ReadFromParts(vm, &testStruct)

	if testStruct.Val != 15 {
		t.Errorf("field value didn't matched got (%d) expected (%d)", testStruct.Val, 15)
		return
	}
}

func TestArrayResolveAssign(t *testing.T) {
	type TestStruct struct {
		Val []int `parts:"arr"`
	}

	vm, err := GetVMWithSource(`let arr = [10, 20]
		arr[0] = 15`)

	if err != nil {
		t.Error(err)
		return
	}

	err = vm.Run()

	if err != nil {
		t.Error(err)
		return
	}

	var testStruct TestStruct

	ReadFromParts(vm, &testStruct)

	if testStruct.Val[0] != 15 {
		t.Errorf("field value didn't matched got (%d) expected (%d)", testStruct.Val[0], 15)
		return
	}

	if testStruct.Val[1] != 20 {
		t.Errorf("field value didn't matched got (%d) expected (%d)", testStruct.Val[1], 20)
		return
	}
}

func TestArrayAssign(t *testing.T) {
	type TestStruct struct {
		Val []int `parts:"arr"`
	}

	vm, err := GetVMWithSource(`let arr = [10, 20]
		arr.0 = 15`)

	if err != nil {
		t.Error(err)
		return
	}

	err = vm.Run()

	if err != nil {
		t.Error(err)
		return
	}

	var testStruct TestStruct

	ReadFromParts(vm, &testStruct)

	if testStruct.Val[0] != 15 {
		t.Errorf("field value didn't matched got (%d) expected (%d)", testStruct.Val[0], 15)
		return
	}

	if testStruct.Val[1] != 20 {
		t.Errorf("field value didn't matched got (%d) expected (%d)", testStruct.Val[1], 20)
		return
	}
}

func TestObjAssign(t *testing.T) {
	type TestStruct struct {
		Val struct {
			Field int `parts:"field"`
		} `parts:"obj"`
	}

	vm, err := GetVMWithSource(`let obj = |> field: 10 <|
		obj.field = 15`)

	if err != nil {
		t.Error(err)
		return
	}

	err = vm.Run()

	if err != nil {
		t.Error(err)
		return
	}

	var testStruct TestStruct

	ReadFromParts(vm, &testStruct)

	if testStruct.Val.Field != 15 {
		t.Errorf("field value didn't matched got (%d) expected (%d)", testStruct.Val.Field, 15)
		return
	}
}

func TestObjResolveAssign(t *testing.T) {
	type TestStruct struct {
		Res int `parts:"res"`
	}

	vm, err := GetVMWithSource(`let key = "field" 
		let obj = |> "field": 10 <|
		obj[key] = 15
		let res = obj[key]`)

	if err != nil {
		t.Error(err)
		return
	}

	err = vm.Run()

	if err != nil {
		t.Error(err)
		return
	}

	var testStruct TestStruct

	ReadFromParts(vm, &testStruct)

	if testStruct.Res != 15 {
		t.Errorf("field value didn't matched got (%d) expected (%d)", testStruct.Res, 15)
		return
	}
}

func TestFunctionCall(t *testing.T) {
	type TestStruct struct {
		Res int `parts:"res"`
	}

	vm, err := GetVMWithSource(`fun GetRes(obj) = obj.res
		let math = |> res: 10 <|
		let res = GetRes(math)`)

	if err != nil {
		t.Error(err)
		return
	}

	err = vm.Run()

	if err != nil {
		t.Error(err)
		return
	}

	var testStruct TestStruct

	ReadFromParts(vm, &testStruct)

	if testStruct.Res != 10 {
		t.Errorf("field value didn't matched got (%d) expected (%d)", testStruct.Res, 10)
		return
	}
}

func TestIfFull(t *testing.T) {
	type TestStruct struct {
		Res int `parts:"res"`
	}

	vm, err := GetVMWithSource(`let res = if (true) {return 10} else {return 1}`)

	if err != nil {
		t.Error(err)
		return
	}

	err = vm.Run()

	if err != nil {
		t.Error(err)
		return
	}

	var testStruct TestStruct

	ReadFromParts(vm, &testStruct)

	if testStruct.Res != 10 {
		t.Errorf("field value didn't matched got (%d) expected (%d)", testStruct.Res, 10)
		return
	}
}

func TestIfFullCursed(t *testing.T) {
	type TestStruct struct {
		Res int `parts:"res"`
	}

	vm, err := GetVMWithSource(`let res = if true return 10 else return 1`)

	if err != nil {
		t.Error(err)
		return
	}

	err = vm.Run()

	if err != nil {
		t.Error(err)
		return
	}

	var testStruct TestStruct

	ReadFromParts(vm, &testStruct)

	if testStruct.Res != 10 {
		t.Errorf("field value didn't matched got (%d) expected (%d)", testStruct.Res, 10)
		return
	}
}

func TestIfNoElse(t *testing.T) {
	type TestStruct struct {
		Res int `parts:"res"`
	}

	vm, err := GetVMWithSource(`let res = if (false) {return 10}`)

	if err != nil {
		t.Error(err)
		return
	}

	err = vm.Run()

	if err == nil {
		t.Error(errors.New("expeced error but got none"))
		return
	}

	if errs, ok := err.(interface{ Unwrap() []error }); ok {
		unwraped := errs.Unwrap()

		if unwraped[len(unwraped)-1].Error() != "got no value, expected value at 'res'" {
			t.Error(errors.New("expected diffrent kind of errror"))
			return
		}
	} else {
		println(err)
		t.Error(errors.New("expected diffrent kind of errror"))
		return
	}
}
