package main

import (
	"fmt"
	"testing"
)

func GetParserWithSource(source string) Parser {
	scanner := GetScannerWithSource(source)

	return Parser{
		Scanner:   &scanner,
		Scope:     TopLevel,
		Literals:  InitialLiterals,
		LastToken: Token{Type: TokenInvalid},
		Meta:      make(map[string]string),
	}
}

func TestLetFalse(t *testing.T) {
	parser := GetParserWithSource("let x = false;")

	bytecode, err := parser.next()

	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}

	expectedBytecode := []Bytecode{
		B_DECLARE, DECLARE_LET, Bytecode(len(InitialLiterals)), 0,
	}

	CheckBytecode(t, bytecode, expectedBytecode)
}

func TestLetTrue(t *testing.T) {
	parser := GetParserWithSource("let x = true;")

	bytecode, err := parser.next()

	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}

	expectedBytecode := []Bytecode{
		B_DECLARE, DECLARE_LET, Bytecode(len(InitialLiterals)), 1,
	}

	CheckBytecode(t, bytecode, expectedBytecode)
}

func TestLetNumber(t *testing.T) {
	parser := GetParserWithSource("let x = 123;")

	bytecode, err := parser.next()

	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}

	expectedBytecode := []Bytecode{
		B_DECLARE, DECLARE_LET, Bytecode(len(InitialLiterals)), Bytecode(len(InitialLiterals)) + 1,
	}

	CheckBytecode(t, bytecode, expectedBytecode)
}

func TestLetString(t *testing.T) {
	parser := GetParserWithSource("let x = \"f\";")

	bytecode, err := parser.next()

	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}

	expectedBytecode := []Bytecode{
		B_DECLARE, DECLARE_LET, Bytecode(len(InitialLiterals)), Bytecode(len(InitialLiterals)) + 1,
	}

	CheckBytecode(t, bytecode, expectedBytecode)
}

func TestParenthesis(t *testing.T) {
	parser := GetParserWithSource("(1);")

	bytecode, err := parser.next()

	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}

	CheckBytecode(t, bytecode, []Bytecode{Bytecode(len(InitialLiterals))})
}

func TestBlock(t *testing.T) {
	parser := GetParserWithSource("{0};")

	bytecode, err := parser.next()

	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}

	CheckBytecode(t, bytecode, []Bytecode{B_NEW_SCOPE, Bytecode(len(InitialLiterals)), B_END_SCOPE})
}

func TestAnonymousFunctionNoBody(t *testing.T) {
	parser := GetParserWithSource("fun () {}")

	bytecode, err := parser.next()

	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}

	CheckBytecode(t, bytecode, []Bytecode{Bytecode(len(InitialLiterals))})
}

func TestAnonymousFunctionWithBody(t *testing.T) {
	parser := GetParserWithSource("fun () { 0 }")

	bytecode, err := parser.next()

	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}

	checkResult := CheckBytecode(t, bytecode, []Bytecode{Bytecode(len(InitialLiterals) + 1)})

	if !checkResult {
		return
	} else {
		t.Log("Got through first check")
	}

	CheckBytecode(t, (parser.Literals[len(parser.Literals)-1].Value).(FunctionDeclaration).Body, []Bytecode{B_NEW_SCOPE, Bytecode(len(InitialLiterals)), B_END_SCOPE})
}

func TestNamedFunctionNoBody(t *testing.T) {
	parser := GetParserWithSource("fun x() {}")

	bytecode, err := parser.next()

	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}

	CheckBytecode(t, bytecode, []Bytecode{B_DECLARE, DECLARE_FUN, Bytecode(len(InitialLiterals)), Bytecode(len(InitialLiterals) + 1)})
}

func TestNamedFunctionWithBody(t *testing.T) {
	parser := GetParserWithSource("fun x() { 0 }")

	bytecode, err := parser.next()

	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}

	checkResult := CheckBytecode(t, bytecode, []Bytecode{B_DECLARE, DECLARE_FUN, Bytecode(len(InitialLiterals)), Bytecode(len(InitialLiterals) + 2)})

	if !checkResult {
		return
	} else {
		t.Log("Got through first check")
	}

	CheckBytecode(t, (parser.Literals[len(parser.Literals)-1].Value).(FunctionDeclaration).Body, []Bytecode{B_NEW_SCOPE, Bytecode(len(InitialLiterals) + 1), B_END_SCOPE})
}

func TestNamedFunctionWithArg(t *testing.T) {
	parser := GetParserWithSource("fun x(one) { }")

	bytecode, err := parser.next()

	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}

	checkResult := CheckBytecode(t, bytecode, []Bytecode{B_DECLARE, DECLARE_FUN, Bytecode(len(InitialLiterals)), Bytecode(len(InitialLiterals) + 1)})

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

	bytecode, err := parser.next()

	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}

	checkResult := CheckBytecode(t, bytecode, []Bytecode{B_DECLARE, DECLARE_FUN, Bytecode(len(InitialLiterals)), Bytecode(len(InitialLiterals) + 1)})

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

func TestObjectNoEntires(t *testing.T) {
	parser := GetParserWithSource("|> <|")

	bytecode, err := parser.next()

	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}

	CheckBytecode(t, bytecode, []Bytecode{Bytecode(len(InitialLiterals))})
}

func TestObjectWithIntEntry(t *testing.T) {
	parser := GetParserWithSource("|> 1 : 0 <|")

	_, err := parser.next()

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

	CheckBytecode(t, objDef[0], append(append(encodedOneOffset, B_SPACING), encodedZeroOffset...))
}

func TestMeta(t *testing.T) {
	parser := GetParserWithSource("#>\"random\": \"value\"")

	bytecode, err := parser.next()

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

	bytecode, err := parser.next()

	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}

	test := CheckBytecode(t, bytecode, []Bytecode{Bytecode(len(InitialLiterals) + 2)})

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

	test = CheckBytecode(t, arr.Entries[0], mustEncodeLen(strIdx))

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

	test = CheckBytecode(t, arr.Entries[1], mustEncodeLen(intIdx))

	if !test {
		return
	}
}

func TestDotExpression(t *testing.T) {
	parser := GetParserWithSource("val.key")

	bytecode, err := parser.next()

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

	CheckBytecode(t, bytecode, append(append(append([]Bytecode{B_DOT}, mustEncodeLen(idxVal)...), B_SPACING), mustEncodeLen(idxKey)...))
}

func TestDotNestedExpression(t *testing.T) {
	parser := GetParserWithSource("(obj.val).key")

	bytecode, err := parser.next()

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

	CheckBytecode(t, bytecode, append(append(append(append(append([]Bytecode{B_DOT, B_DOT}, mustEncodeLen(idxObj)...), B_SPACING), mustEncodeLen(idxVal)...), B_SPACING), mustEncodeLen(idxKey)...))
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

		if literal.Value != val && val != nil {
			continue
		}

		return idx, literal
	}

	return -1, Literal{}
}
