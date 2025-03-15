package parts

type TokenType = int

var Operators = []rune{'+', '-', '/', '*', ';', '[', ']', '(', ')', '{', '}', '.', ':', ',', '|', '&', '>', '<', '!', '#', '-', '=', '?'}

var Keywords = []string{
	"false", "if", "let", "true",
	"fun", "return", "else", "static",
	"for", "class", "break", "continue",
	"import", "from", "as",
}

var ValidOperators = map[string]string{
	"+":  "PLUS",
	"-":  "MINUS",
	"/":  "SLASH",
	"*":  "STAR",
	";":  "SEMICOLON",
	":":  "COLON",
	".":  "DOT",
	",":  "COMMA",
	"(":  "LEFT_PAREN",
	")":  "RIGHT_PAREN",
	"{":  "LEFT_BRACE",
	"}":  "RIGHT_BRACE",
	"[":  "LEFT_BRACKET",
	"]":  "RIGHT_BRACKET",
	"@":  "AT",
	"=":  "EQUALS",
	"|>": "OBJ_START",
	"<|": "OBJ_END",
	"#>": "META",
	"==": "EQUALITY",
}

const (
	TokenOperator TokenType = iota
	TokenNumber
	TokenKeyword
	TokenIdentifier
	TokenString
	TokenNewLine
	TokenInvalid
)

type Rule struct {
	Result   TokenType
	BaseRule func(r rune) bool
}

var ScannerRules = []Rule{
	{
		Result: TokenOperator,
		BaseRule: func(r rune) bool {
			for _, v := range Operators {
				if r == v {
					return true
				}
			}
			return false
		},
	},
	{
		Result: TokenNumber,
		BaseRule: func(r rune) bool {
			return r >= '0' && r <= '9'
		},
	},
	{
		Result: TokenKeyword,
		BaseRule: func(r rune) bool {
			return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
		},
	},
	{
		Result: TokenNewLine,
		BaseRule: func(r rune) bool {
			return r == '\n' || r == '\r'
		},
	},
}

type Token struct {
	Type  TokenType
	Value []rune
}
