package parts

type Scope int

const (
	TopLevel Scope = iota
	Expression
)

type Parser struct {
	Scanner *Scanner
	Scope   Scope
}
