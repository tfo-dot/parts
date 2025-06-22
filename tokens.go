package parts

type TokenType = int

const (
	TokenInvalid TokenType = iota
	TokenOperator 
	TokenNumber
	TokenKeyword
	TokenIdentifier
	TokenString
	TokenSpace
)

type Token struct {
	Type  TokenType
	Value []rune
}
