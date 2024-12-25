package parts

type TokenType = int

var Operators = []rune{'+', '-', '/', '*', ';', '[', ']', '(', ')', '{', '}', '.', ':', ',', '|', '&', '>', '<', '!', '#', '-', '=', '?'}

const (
	TokenOperator TokenType = iota
	TokenNumber
	TokenKeyword
	TokenString
	TokenNewLine
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
