package parts

import "testing"

func GetScannerWithSource(source string) Scanner {
	return Scanner{Source: []rune(source), Rules: ScannerRules}
}

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
