package parts

import "testing"

func TestScanner(t *testing.T) {
	scanner := GetScannerWithSource("\"hello\"")

	token, err := scanner.Next()

	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	if string(token.Value) != "hello" {
		t.Errorf("unexpected token value: %s", string(token.Value))
	}

	if token.Type != TokenString {
		t.Errorf("unexpected token type: %d", token.Type)
	}
}

func TestScannerUnterminatedString(t *testing.T) {
	scanner := GetScannerWithSource("\"hello")

	_, err := scanner.Next()

	if err == nil {
		t.Errorf("expected error")
	}
}

func TestScannerUnknownToken(t *testing.T) {
	scanner := GetScannerWithSource("👋")

	_, err := scanner.Next()

	if err == nil {
		t.Errorf("expected error")
	}
}

func TestScannerLetStatement(t *testing.T) {
	scanner := GetScannerWithSource("let x = 0;")

	expectedTokens := []Token{
		{Type: TokenKeyword, Value: []rune("LET")},
		{Type: TokenIdentifier, Value: []rune("x")},
		{Type: TokenOperator, Value: []rune("EQUALS")},
		{Type: TokenNumber, Value: []rune("0")},
		{Type: TokenOperator, Value: []rune("SEMICOLON")},
	}

	for _, curr := range expectedTokens {
		token, err := scanner.Next()

		if err != nil {
			t.Errorf("unexpected error: %s", err)
			return
		}

		if string(token.Value) != string(curr.Value) {
			t.Errorf("token values don't match: %s != %s", string(token.Value), string(curr.Value))
			return
		}

		if token.Type != curr.Type {
			t.Errorf("token types don't match: %d != %d", token.Type, curr.Type)
			return
		}
	}
}

func TestEmptyParen(t *testing.T) {
	scanner := GetScannerWithSource("();")

	expectedTokens := []Token{
		{Type: TokenOperator, Value: []rune("LEFT_PAREN")},
		{Type: TokenOperator, Value: []rune("RIGHT_PAREN")},
		{Type: TokenOperator, Value: []rune("SEMICOLON")},
	}

	for _, curr := range expectedTokens {
		token, err := scanner.Next()

		if err != nil {
			t.Errorf("unexpected error: %s", err)
			return
		}

		if string(token.Value) != string(curr.Value) {
			t.Errorf("token values don't match: %s != %s", string(token.Value), string(curr.Value))
			return
		}

		if token.Type != curr.Type {
			t.Errorf("token types don't match: %d != %d", token.Type, curr.Type)
			return
		}
	}
}

func TestString(t *testing.T) {
	scanner := GetScannerWithSource("\"text\" + \"text\"")

	expectedTokens := []Token{
		{Type: TokenString, Value: []rune("text")},
		{Type: TokenOperator, Value: []rune("PLUS")},
		{Type: TokenString, Value: []rune("text")},
	}

	for _, curr := range expectedTokens {
		token, err := scanner.Next()

		if err != nil {
			t.Errorf("unexpected error: %s", err)
			return
		}

		if string(token.Value) != string(curr.Value) {
			t.Errorf("token values don't match: %s != %s", string(token.Value), string(curr.Value))
			return
		}

		if token.Type != curr.Type {
			t.Errorf("token types don't match: %d != %d", token.Type, curr.Type)
			return
		}
	}
}

func TestModdedScanner(t *testing.T) {
	scanner := GetScannerWithSource("let x |= 0")

	for _, rule  := range scanner.Rules {
		if rule.Result == TokenOperator {
			rule.Mappings["|="] = "PIPE_EQ"

			break
		}
	}

	expectedTokens := []Token{
		{Type: TokenKeyword, Value: []rune("LET")},
		{Type: TokenIdentifier, Value: []rune("x")},
		{Type: TokenOperator, Value: []rune("PIPE_EQ")},
		{Type: TokenNumber, Value: []rune("0")},
	}

	for _, curr := range expectedTokens {
		token, err := scanner.Next()

		if err != nil {
			t.Errorf("unexpected error: %s", err)
			return
		}

		if string(token.Value) != string(curr.Value) {
			t.Errorf("token values don't match: %s != %s", string(token.Value), string(curr.Value))
			return
		}

		if token.Type != curr.Type {
			t.Errorf("token types don't match: %d != %d", token.Type, curr.Type)
			return
		}
	}
}
