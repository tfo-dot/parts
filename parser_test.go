package parts

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

func CheckBytecode(t *testing.T, result []Bytecode, expected []Bytecode) bool {
	fmt.Printf("%v ?? %v\n", result, expected)

	for idx, val := range expected {
		t.Logf("checking bytecode %d ?? %d", val, result[idx])
		if val != result[idx] {
			t.Error("bytecode don't match")
			return false
		}
	}

	return true
}
