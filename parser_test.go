package parts

import "testing"

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

func CheckBytecode(t *testing.T, result []Bytecode, expected []Bytecode) {
	expectedBytecode := []Bytecode{Bytecode(len(InitialLiterals))}

	for idx, val := range expectedBytecode {
		t.Logf("checking bytecode %d ?? %d", val, result[idx])
		if val != result[idx] {
			t.Errorf("bytecode don't match %d != %d", val, result[idx])
			return
		}
	}
}
