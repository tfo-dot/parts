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

func TestDot(t *testing.T) {
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