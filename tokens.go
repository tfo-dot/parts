package parts

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

type Token struct {
	Type  TokenType
	Value []rune
}
