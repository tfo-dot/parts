package parts

import (
	"fmt"
	"testing"
)

func TestLetFalse(t *testing.T) {
	parser := GetParserWithSource("let x = false;")

	bytecode, err := parser.parse()

	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}

	CheckBytecode(t, bytecode, []Bytecode{
		B_DECLARE, B_LITERAL, Bytecode(len(InitialLiterals)), 0,
	})
}

func TestLetTrue(t *testing.T) {
	parser := GetParserWithSource("let x = true;")

	bytecode, err := parser.parse()

	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}

	CheckBytecode(t, bytecode, []Bytecode{
		B_DECLARE, B_LITERAL, Bytecode(len(InitialLiterals)), 1,
	})
}

func TestLetNumber(t *testing.T) {
	parser := GetParserWithSource("let x = 123;")

	bytecode, err := parser.parse()

	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}

	expectedBytecode := []Bytecode{
		B_DECLARE, B_LITERAL, Bytecode(len(InitialLiterals)), B_LITERAL, Bytecode(len(InitialLiterals)) + 1,
	}

	CheckBytecode(t, bytecode, expectedBytecode)
}

func TestLetString(t *testing.T) {
	parser := GetParserWithSource("let x = \"f\";")

	bytecode, err := parser.parse()

	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}

	expectedBytecode := []Bytecode{
		B_DECLARE, B_LITERAL, Bytecode(len(InitialLiterals)), B_LITERAL, Bytecode(len(InitialLiterals)) + 1,
	}

	CheckBytecode(t, bytecode, expectedBytecode)
}

func TestParenthesis(t *testing.T) {
	parser := GetParserWithSource("(1);")

	bytecode, err := parser.parse()

	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}

	CheckBytecode(t, bytecode, []Bytecode{B_LITERAL, Bytecode(len(InitialLiterals))})
}

func TestBlock(t *testing.T) {
	parser := GetParserWithSource("{0};")

	bytecode, err := parser.parse()

	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}

	CheckBytecode(t, bytecode, []Bytecode{B_NEW_SCOPE, B_LITERAL, Bytecode(len(InitialLiterals)), B_END_SCOPE})
}

func TestAnonymousFunctionNoBody(t *testing.T) {
	parser := GetParserWithSource("fun () {}")

	bytecode, err := parser.parse()

	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}

	CheckBytecode(t, bytecode, []Bytecode{B_LITERAL, Bytecode(len(InitialLiterals))})
}

func TestAnonymousFunctionWithBody(t *testing.T) {
	parser := GetParserWithSource("fun () { 0 }")

	bytecode, err := parser.parse()

	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}

	checkResult := CheckBytecode(t, bytecode, []Bytecode{B_LITERAL, Bytecode(len(InitialLiterals) + 1)})

	if !checkResult {
		return
	} else {
		t.Log("Got through first check")
	}

	CheckBytecode(t, (parser.Literals[len(parser.Literals)-1].Value).(FunctionDeclaration).Body, []Bytecode{B_NEW_SCOPE, B_LITERAL, Bytecode(len(InitialLiterals)), B_END_SCOPE})
}

func TestNamedFunctionNoBody(t *testing.T) {
	parser := GetParserWithSource("fun x() {}")

	bytecode, err := parser.parse()

	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}

	CheckBytecode(t, bytecode, []Bytecode{B_DECLARE, B_LITERAL, Bytecode(len(InitialLiterals)), B_LITERAL, Bytecode(len(InitialLiterals) + 1)})
}

func TestNamedFunctionWithBody(t *testing.T) {
	parser := GetParserWithSource("fun x() { 0 }")

	bytecode, err := parser.parse()

	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}

	checkResult := CheckBytecode(t, bytecode, []Bytecode{B_DECLARE, B_LITERAL, Bytecode(len(InitialLiterals)), B_LITERAL, Bytecode(len(InitialLiterals) + 2)})

	if !checkResult {
		return
	} else {
		t.Log("Got through first check")
	}

	CheckBytecode(t, (parser.Literals[len(parser.Literals)-1].Value).(FunctionDeclaration).Body, []Bytecode{B_NEW_SCOPE, B_LITERAL, Bytecode(len(InitialLiterals) + 1), B_END_SCOPE})
}

func TestNamedFunctionWithArg(t *testing.T) {
	parser := GetParserWithSource("fun x(one) { }")

	bytecode, err := parser.parse()

	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}

	checkResult := CheckBytecode(t, bytecode, []Bytecode{B_DECLARE, B_LITERAL, Bytecode(len(InitialLiterals)), B_LITERAL, Bytecode(len(InitialLiterals) + 1)})

	if !checkResult {
		return
	} else {
		t.Log("Got through first check")
	}

	fnDeclaration := (parser.Literals[len(parser.Literals)-1].Value).(FunctionDeclaration)

	if len(fnDeclaration.Params) != 1 {
		t.Errorf("expected 1 param got %d", len(fnDeclaration.Params))
		return
	}

	if string(fnDeclaration.Params[0]) != "one" {
		t.Errorf("expected 'one' at [0] in declaration got %s", string(fnDeclaration.Params[0]))
		return
	}
}

func TestNamedFunctionWithTwoArg(t *testing.T) {
	parser := GetParserWithSource("fun x(one, two) { }")

	bytecode, err := parser.parse()

	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}

	checkResult := CheckBytecode(t, bytecode, []Bytecode{B_DECLARE, B_LITERAL, Bytecode(len(InitialLiterals)), B_LITERAL, Bytecode(len(InitialLiterals) + 1)})

	if !checkResult {
		return
	} else {
		t.Log("Got through first check")
	}

	fnDeclaration := (parser.Literals[len(parser.Literals)-1].Value).(FunctionDeclaration)

	if len(fnDeclaration.Params) != 2 {
		t.Errorf("expected 2 params got %d", len(fnDeclaration.Params))
		return
	}

	if string(fnDeclaration.Params[0]) != "one" {
		t.Errorf("expected 'one' at [0] in declaration got %s", string(fnDeclaration.Params[0]))
		return
	}

	if string(fnDeclaration.Params[1]) != "two" {
		t.Errorf("expected 'two' at [1] in declaration got %s", string(fnDeclaration.Params[1]))
		return
	}
}

func TestNamedFunctionInline(t *testing.T) {
	parser := GetParserWithSource("fun x(one) = one")

	bytecode, err := parser.parse()

	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}

	oneIdx, _ := GetParserLiteral(parser, RefLiteral, "one")

	if oneIdx == -1 {
		t.Error("literal (Ref, 'one') wasn't present")
	}

	idx, _ := GetParserLiteral(parser, RefLiteral, "x")

	if idx == -1 {
		t.Error("literal (Ref, 'x') wasn't present")
	}

	funcIdx, _ := GetParserLiteral(parser, FunLiteral, nil)

	if funcIdx == -1 {
		t.Error("literal (Fun) wasn't present")
	}

	checkResult := CheckBytecode(t, bytecode, []Bytecode{B_DECLARE, B_LITERAL, Bytecode(idx), B_LITERAL, Bytecode(funcIdx)})

	if !checkResult {
		return
	} else {
		t.Log("Got through first check")
	}

	fnDeclaration := (parser.Literals[len(parser.Literals)-1].Value).(FunctionDeclaration)

	if len(fnDeclaration.Params) != 1 {
		t.Errorf("expected 2 params got %d", len(fnDeclaration.Params))
		return
	}

	if string(fnDeclaration.Params[0]) != "one" {
		t.Errorf("expected 'one' at [0] in declaration got %s", string(fnDeclaration.Params[0]))
		return
	}

	CheckBytecode(t, fnDeclaration.Body, []Bytecode{B_RETURN, B_LITERAL, Bytecode(oneIdx)})
}

func TestObjectDeclaration(t *testing.T) {
	parser := GetParserWithSource("let x = |> <|")

	bytecode, err := parser.parse()

	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}

	expectedBytecode := []Bytecode{
		B_DECLARE, B_LITERAL, Bytecode(len(InitialLiterals)), B_LITERAL, Bytecode(len(InitialLiterals)) + 1,
	}

	CheckBytecode(t, bytecode, expectedBytecode)
}

func TestObjectNoEntires(t *testing.T) {
	parser := GetParserWithSource("|> <|")

	bytecode, err := parser.parse()

	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}

	CheckBytecode(t, bytecode, []Bytecode{B_LITERAL, Bytecode(len(InitialLiterals))})
}

func TestObjectWithIntEntry(t *testing.T) {
	parser := GetParserWithSource("|> 1 : 0 <|")

	_, err := parser.parse()

	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}

	idxObj, objRaw := GetParserLiteral(parser, ObjLiteral, nil)

	if idxObj == -1 {
		t.Error("object literal wasn't present")
		return
	}

	objDef := objRaw.Value.(ObjDefinition).Entries

	if len(objDef) != 1 {
		t.Errorf("entires number doesn't match (%d ?? %d)", len(objDef), 1)
		return
	}

	idxZero, _ := GetParserLiteral(parser, IntLiteral, 0)

	if idxZero == -1 {
		t.Error("literal (Int, 0) wasn't present")
		return
	}

	encodedZeroOffset, err := encodeLen(idxZero)

	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}

	idxOne, _ := GetParserLiteral(parser, IntLiteral, 1)

	if idxOne == -1 {
		t.Error("literal (Int, 1) wasn't present")
		return
	}

	encodedOneOffset, err := encodeLen(idxOne)

	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}

	CheckBytecode(t, objDef[0], append(append(append([]Bytecode{B_LITERAL}, encodedOneOffset...), B_LITERAL), encodedZeroOffset...))
}

func TestMeta(t *testing.T) {
	parser := GetParserWithSource("#>\"random\": \"value\"")

	bytecode, err := parser.parse()

	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}

	if len(bytecode) != 0 {
		t.Error("bytecode is not empty")
		return
	}

	if parser.Meta["random"] != "value" {
		t.Errorf("meta values don't match %s != \"value\"", parser.Meta["random"])
		return
	}
}

func TestArray(t *testing.T) {
	parser := GetParserWithSource("[\"a\", 1]")

	bytecode, err := parser.parse()

	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}

	test := CheckBytecode(t, bytecode, []Bytecode{B_LITERAL, Bytecode(len(InitialLiterals) + 2)})

	if !test {
		return
	}

	arr := parser.Literals[len(InitialLiterals)+2].Value.(ListDefinition)

	if len(arr.Entries) != 2 {
		t.Error("array has invalid length")
		return
	}

	strIdx, strVal := GetParserLiteral(parser, StringLiteral, nil)

	if strIdx == -1 {
		t.Error("literal (Str, \"a\") wasn't present")
		return
	}

	if strVal.Value.(string) != "a" {
		t.Errorf("literal (Str, \"a\") values didn't match (%s)", strVal.Value)
		return
	}

	test = CheckBytecode(t, arr.Entries[0], append([]Bytecode{B_LITERAL}, mustEncodeLen(strIdx)...))

	if !test {
		return
	}

	intIdx, intVal := GetParserLiteral(parser, IntLiteral, nil)

	if intIdx == -1 {
		t.Error("literal (Int, 1) wasn't present")
		return
	}

	if intVal.Value.(int) != 1 {
		t.Errorf("literal (Int, 1) values didn't match (%s)", intVal.Value)
		return
	}

	test = CheckBytecode(t, arr.Entries[1], append([]Bytecode{B_LITERAL}, mustEncodeLen(intIdx)...))

	if !test {
		return
	}
}

func TestDotExpression(t *testing.T) {
	parser := GetParserWithSource("val.key")

	bytecode, err := parser.parse()

	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}

	idxVal, _ := GetParserLiteral(parser, RefLiteral, "val")

	if idxVal == -1 {
		t.Error("literal (Ref, 'val') wasn't present")
		return
	}

	idxKey, _ := GetParserLiteral(parser, RefLiteral, "key")

	if idxKey == -1 {
		t.Error("literal (Ref, 'key') wasn't present")
		return
	}

	CheckBytecode(t, bytecode, append(append(append([]Bytecode{B_DOT, B_LITERAL}, mustEncodeLen(idxVal)...), B_LITERAL), mustEncodeLen(idxKey)...))
}

func TestDotNestedExpression(t *testing.T) {
	parser := GetParserWithSource("(obj.val).key")

	bytecode, err := parser.parse()

	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}

	idxObj, _ := GetParserLiteral(parser, RefLiteral, "obj")

	if idxObj == -1 {
		t.Error("literal (Ref, 'obj') wasn't present")
		return
	}

	idxVal, _ := GetParserLiteral(parser, RefLiteral, "val")

	if idxVal == -1 {
		t.Error("literal (Ref, 'val') wasn't present")
		return
	}

	idxKey, _ := GetParserLiteral(parser, RefLiteral, "key")

	if idxKey == -1 {
		t.Error("literal (Ref, 'key') wasn't present")
		return
	}

	CheckBytecode(t, bytecode, append(append(append(append(append([]Bytecode{B_DOT, B_DOT, B_LITERAL}, mustEncodeLen(idxObj)...), B_LITERAL), mustEncodeLen(idxVal)...), B_LITERAL), mustEncodeLen(idxKey)...))
}

func TestRefListAccess(t *testing.T) {
	parser := GetParserWithSource("x[0]")

	bytecode, err := parser.parse()

	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}

	varKey, _ := GetParserLiteral(parser, RefLiteral, "x")

	if varKey == -1 {
		t.Error("literal (Ref, 'x') wasn't present")
		return
	}

	idxKey, _ := GetParserLiteral(parser, IntLiteral, 0)

	if idxKey == -1 {
		t.Error("literal (Int, 0) wasn't present")
		return
	}

	CheckBytecode(t, bytecode, append([]Bytecode{B_DOT, B_LITERAL}, Bytecode(varKey), B_RESOLVE, B_LITERAL, Bytecode(idxKey)))
}

func TestFunCalSingleArg(t *testing.T) {
	parser := GetParserWithSource("x(10)")

	bytecode, err := parser.parse()

	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}

	varKey, _ := GetParserLiteral(parser, RefLiteral, "x")

	if varKey == -1 {
		t.Error("literal (Ref, 'x') wasn't present")
		return
	}

	idxKey, _ := GetParserLiteral(parser, IntLiteral, 10)

	if idxKey == -1 {
		t.Error("literal (Int, 10) wasn't present")
		return
	}

	CheckBytecode(t, bytecode, append([]Bytecode{B_CALL, B_LITERAL}, Bytecode(varKey), Bytecode(1), B_LITERAL, Bytecode(idxKey)))
}

func TestFunCallAssign(t *testing.T) {
	parser := GetParserWithSource("let y = x(10)")

	bytecode, err := parser.parse()

	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}

	varKey, _ := GetParserLiteral(parser, RefLiteral, "x")

	if varKey == -1 {
		t.Error("literal (Ref, 'x') wasn't present")
		return
	}

	varKey2, _ := GetParserLiteral(parser, RefLiteral, "y")

	if varKey2 == -1 {
		t.Error("literal (Ref, 'y') wasn't present")
		return
	}

	idxKey, _ := GetParserLiteral(parser, IntLiteral, 10)

	if idxKey == -1 {
		t.Error("literal (Int, 10) wasn't present")
		return
	}

	CheckBytecode(t, bytecode, []Bytecode{B_DECLARE, B_LITERAL, Bytecode(varKey2), B_CALL, B_LITERAL, Bytecode(varKey), 1, B_LITERAL, Bytecode(idxKey)})
}

func TestFunCalMultipleArgs(t *testing.T) {
	parser := GetParserWithSource("x(10, 20)")

	bytecode, err := parser.parse()

	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}

	varKey, _ := GetParserLiteral(parser, RefLiteral, "x")

	if varKey == -1 {
		t.Error("literal (Ref, 'x') wasn't present")
		return
	}

	idxKey, _ := GetParserLiteral(parser, IntLiteral, 10)

	if idxKey == -1 {
		t.Error("literal (Int, 10) wasn't present")
		return
	}

	idxKey2, _ := GetParserLiteral(parser, IntLiteral, 20)

	if idxKey == -1 {
		t.Error("literal (Int, 20) wasn't present")
		return
	}

	CheckBytecode(t, bytecode, append([]Bytecode{B_CALL, B_LITERAL}, Bytecode(varKey), Bytecode(2), B_LITERAL, Bytecode(idxKey), B_LITERAL, Bytecode(idxKey2)))
}

func TestFunFieldCal(t *testing.T) {
	parser := GetParserWithSource("x.y(10)")

	bytecode, err := parser.parse()

	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}

	varKey, _ := GetParserLiteral(parser, RefLiteral, "x")

	if varKey == -1 {
		t.Error("literal (Ref, 'x') wasn't present")
		return
	}

	varKey2, _ := GetParserLiteral(parser, RefLiteral, "y")

	if varKey2 == -1 {
		t.Error("literal (Ref, 'y') wasn't present")
		return
	}

	varVal, _ := GetParserLiteral(parser, IntLiteral, 10)

	if varVal == -1 {
		t.Error("literal (Int, 10) wasn't present")
		return
	}

	CheckBytecode(t, bytecode, []Bytecode{B_CALL, B_DOT, B_LITERAL, Bytecode(varKey), B_LITERAL, Bytecode(varKey2), Bytecode(1), B_LITERAL, Bytecode(varVal)})
}

func TestFunFieldArrCal(t *testing.T) {
	parser := GetParserWithSource("x[y](10)")

	bytecode, err := parser.parse()

	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}

	varKey, _ := GetParserLiteral(parser, RefLiteral, "x")

	if varKey == -1 {
		t.Error("literal (Ref, 'x') wasn't present")
		return
	}

	varKey2, _ := GetParserLiteral(parser, RefLiteral, "y")

	if varKey2 == -1 {
		t.Error("literal (Ref, 'y') wasn't present")
		return
	}

	varVal, _ := GetParserLiteral(parser, IntLiteral, 10)

	if varVal == -1 {
		t.Error("literal (Int, 10) wasn't present")
		return
	}

	CheckBytecode(t, bytecode, []Bytecode{B_CALL, B_DOT, B_LITERAL, Bytecode(varKey), B_RESOLVE, B_LITERAL, Bytecode(varKey2), Bytecode(1), B_LITERAL, Bytecode(varVal)})
}

func TestSetVarExpression(t *testing.T) {
	parser := GetParserWithSource("x = 0")

	bytecode, err := parser.parseAll()

	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}

	varKey, _ := GetParserLiteral(parser, RefLiteral, "x")

	if varKey == -1 {
		t.Error("literal (Ref, 'x') wasn't present")
		return
	}

	varVal, _ := GetParserLiteral(parser, IntLiteral, 0)

	if varVal == -1 {
		t.Error("literal (Int, 0) wasn't present")
		return
	}

	CheckBytecode(t, bytecode, []Bytecode{B_SET, B_LITERAL, Bytecode(varKey), B_LITERAL, Bytecode(varVal)})
}

func TestSetObjExpression(t *testing.T) {
	parser := GetParserWithSource("obj.key = 10")

	bytecode, err := parser.parse()

	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}

	varKey, _ := GetParserLiteral(parser, RefLiteral, "obj")

	if varKey == -1 {
		t.Error("literal (Ref, 'obj') wasn't present")
		return
	}

	varKey2, _ := GetParserLiteral(parser, RefLiteral, "key")

	if varKey2 == -1 {
		t.Error("literal (Ref, 'key') wasn't present")
		return
	}

	varVal, _ := GetParserLiteral(parser, IntLiteral, 10)

	if varVal == -1 {
		t.Error("literal (Int, 10) wasn't present")
		return
	}

	CheckBytecode(t, bytecode, []Bytecode{B_SET, B_DOT, B_LITERAL, Bytecode(varKey), B_LITERAL, Bytecode(varKey2), B_LITERAL, Bytecode(varVal)})
}

func TestSetObjIndexExpression(t *testing.T) {
	parser := GetParserWithSource("obj[key] = 10")

	bytecode, err := parser.parse()

	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}

	varKey, _ := GetParserLiteral(parser, RefLiteral, "obj")

	if varKey == -1 {
		t.Error("literal (Ref, 'obj') wasn't present")
		return
	}

	varKey2, _ := GetParserLiteral(parser, RefLiteral, "key")

	if varKey2 == -1 {
		t.Error("literal (Ref, 'key') wasn't present")
		return
	}

	varVal, _ := GetParserLiteral(parser, IntLiteral, 10)

	if varVal == -1 {
		t.Error("literal (Int, 10) wasn't present")
		return
	}

	CheckBytecode(t, bytecode, []Bytecode{B_SET, B_DOT, B_LITERAL, Bytecode(varKey), B_RESOLVE, B_LITERAL, Bytecode(varKey2), B_LITERAL, Bytecode(varVal)})
}

func TestSetListExpression(t *testing.T) {
	parser := GetParserWithSource("list[0] = 10")

	bytecode, err := parser.parse()

	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}

	varKey, _ := GetParserLiteral(parser, RefLiteral, "list")

	if varKey == -1 {
		t.Error("literal (Ref, 'x') wasn't present")
		return
	}

	varVal, _ := GetParserLiteral(parser, IntLiteral, 0)

	if varVal == -1 {
		t.Error("literal (Int, 0) wasn't present")
		return
	}

	varVal2, _ := GetParserLiteral(parser, IntLiteral, 10)

	if varVal2 == -1 {
		t.Error("literal (Int, 10) wasn't present")
		return
	}

	CheckBytecode(t, bytecode, []Bytecode{B_SET, B_DOT, B_LITERAL, Bytecode(varKey), B_RESOLVE, B_LITERAL, Bytecode(varVal), B_LITERAL, Bytecode(varVal2)})
}

func TestSetDotListExpression(t *testing.T) {
	parser := GetParserWithSource("list.0 = 10")

	bytecode, err := parser.parse()

	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}

	varKey, _ := GetParserLiteral(parser, RefLiteral, "list")

	if varKey == -1 {
		t.Error("literal (Ref, 'x') wasn't present")
		return
	}

	varVal, _ := GetParserLiteral(parser, IntLiteral, 0)

	if varVal == -1 {
		t.Error("literal (Int, 0) wasn't present")
		return
	}

	varVal2, _ := GetParserLiteral(parser, IntLiteral, 10)

	if varVal2 == -1 {
		t.Error("literal (Int, 10) wasn't present")
		return
	}

	CheckBytecode(t, bytecode, []Bytecode{B_SET, B_DOT, B_LITERAL, Bytecode(varKey), B_LITERAL, Bytecode(varVal), B_LITERAL, Bytecode(varVal2)})
}

func TestSetListDynamicExpression(t *testing.T) {
	parser := GetParserWithSource("list[idx] = 10")

	bytecode, err := parser.parse()

	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}

	varKey, _ := GetParserLiteral(parser, RefLiteral, "list")

	if varKey == -1 {
		t.Error("literal (Ref, 'list') wasn't present")
		return
	}

	varKey2, _ := GetParserLiteral(parser, RefLiteral, "idx")

	if varKey2 == -1 {
		t.Error("literal (Ref, 'idx') wasn't present")
		return
	}

	varVal, _ := GetParserLiteral(parser, IntLiteral, 10)

	if varVal == -1 {
		t.Error("literal (Int, 10) wasn't present")
		return
	}

	CheckBytecode(t, bytecode, []Bytecode{B_SET, B_DOT, B_LITERAL, Bytecode(varKey), B_RESOLVE, B_LITERAL, Bytecode(varKey2), B_LITERAL, Bytecode(varVal)})
}

func TestSetListDotFieldExpression(t *testing.T) {
	parser := GetParserWithSource("list.idx = 10")


	bytecode, err := parser.parse()

	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}

	varKey, _ := GetParserLiteral(parser, RefLiteral, "list")

	if varKey == -1 {
		t.Error("literal (Ref, 'list') wasn't present")
		return
	}

	varKey2, _ := GetParserLiteral(parser, RefLiteral, "idx")

	if varKey2 == -1 {
		t.Error("literal (Ref, 'idx') wasn't present")
		return
	}

	varVal, _ := GetParserLiteral(parser, IntLiteral, 10)

	if varVal == -1 {
		t.Error("literal (Int, 10) wasn't present")
		return
	}

	CheckBytecode(t, bytecode, []Bytecode{B_SET, B_DOT, B_LITERAL, Bytecode(varKey), B_LITERAL, Bytecode(varKey2), B_LITERAL, Bytecode(varVal)})
}

func CheckBytecode(t *testing.T, result []Bytecode, expected []Bytecode) bool {
	fmt.Printf("Checking chunks: %v ?? %v\n", result, expected)

	for idx, val := range expected {
		t.Logf("checking bytecode %d ?? %d", val, result[idx])
		if val != result[idx] {
			t.Error("bytecode don't match")
			return false
		}
	}

	return true
}

func GetParserLiteral(p Parser, literalType LiteralType, val any) (int, Literal) {
	for idx, literal := range p.Literals {
		if literal.LiteralType != literalType {
			continue
		}

		if val == nil {
			return idx, literal
		}

		if literalType == RefLiteral && literal.Value.(ReferenceDeclaration).Reference == val {
			return idx, literal
		}

		if literal.Value != val {
			continue
		}

		return idx, literal
	}

	return -1, Literal{}
}
