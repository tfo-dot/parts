package parts

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
)

type TokenType = int

const (
	TokenOperator TokenType = iota
	TokenNumber
	TokenKeyword
	TokenIdentifier
	TokenString
	TokenSpace
	TokenInvalid
)

type RuleType = int

const (
	SKIP_RULE RuleType = iota
	TOKEN_RULE
)

type Rule struct {
	Result     TokenType
	BaseRule   func(r rune) bool
	Rule       func(runs []rune) bool
	Process    func(mappings map[string]string, runs []rune) ([]Token, error)
	RType      RuleType
	Mappings   map[string]string
	ValidChars []rune
}

var ScannerRules = []Rule{
	{
		Result: TokenOperator,
		RType:  TOKEN_RULE,
		Process: func(mappings map[string]string, runs []rune) ([]Token, error) {
			tokenValue := string(runs)
			name, ok := mappings[tokenValue]

			if ok {
				return []Token{{Type: TokenOperator, Value: []rune(name)}}, nil
			}

			offset := 0
			retTokens := make([]Token, 0)

			temp := tokenValue

			for offset != len(tokenValue) {
				if len(temp) == 0 {
					return []Token{}, fmt.Errorf("not valid operator ( %s )", tokenValue[offset:])
				}

				name, ok := mappings[temp]

				if ok {
					offset += len(temp)

					retTokens = append(retTokens, Token{Type: TokenOperator, Value: []rune(name)})

					temp = tokenValue[offset:]
				} else {
					temp = temp[:len(temp)-1]
				}
			}

			return retTokens, nil
		},
		ValidChars: []rune{'+', '-', '/', '*', ';', '[', ']', '(', ')', '{', '}', '.', ':', ',', '|', '&', '>', '<', '!', '#', '-', '=', '?'},
		Mappings: map[string]string{
			"+": "PLUS", "-": "MINUS", "/": "SLASH", "*": "STAR",
			";": "SEMICOLON", ":": "COLON",
			".": "DOT", ",": "COMMA",
			"(": "LEFT_PAREN", ")": "RIGHT_PAREN",
			"{": "LEFT_BRACE", "}": "RIGHT_BRACE",
			"[": "LEFT_BRACKET", "]": "RIGHT_BRACKET",
			"@": "AT", "=": "EQUALS",
			"|>": "OBJ_START", "<|": "OBJ_END",
			"#>": "META", "==": "EQUALITY",
		},
	},
	{
		Result: TokenNumber,
		BaseRule: func(r rune) bool {
			return r >= '0' && r <= '9'
		},
		RType: TOKEN_RULE,
	},
	{
		Result: TokenKeyword,
		BaseRule: func(r rune) bool {
			return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_'
		},
		Process: func(mappings map[string]string, runs []rune) ([]Token, error) {
			token := Token{
				Type:  TokenKeyword,
				Value: runs,
			}

			if _, has := mappings[string(token.Value)]; has {
				token.Value = []rune(strings.ToUpper(string(token.Value)))
			} else {
				token.Type = TokenIdentifier
			}

			return []Token{token}, nil
		},
		RType: TOKEN_RULE,
		Mappings: map[string]string{
			"false": "", "if": "", "let": "", "true": "",
			"fun": "", "return": "", "else": "", "static": "",
			"for": "", "class": "", "break": "", "continue": "",
			"import": "", "from": "", "as": "", "syntax": "",
		},
	},
	{
		Result: TokenSpace,
		BaseRule: func(r rune) bool {
			return unicode.IsSpace(r)
		},
		RType: SKIP_RULE,
	},
	{
		Result:   TokenString,
		RType:    TOKEN_RULE,
		BaseRule: func(r rune) bool { return true },
		Rule:     func(runs []rune) bool { return len(runs) == 1 || runs[len(runs)-1] != '"' },
		Process: func(mappings map[string]string, runs []rune) ([]Token, error) {
			token := Token{Type: TokenString, Value: runs}

			if runs[0] != '"' || runs[len(runs)-1] != '"' {
				return []Token{}, errors.New("got unterminated string")
			}

			token.Value = token.Value[1 : len(runs)-1]

			return []Token{token}, nil
		},
	},
}

type Token struct {
	Type  TokenType
	Value []rune
}
