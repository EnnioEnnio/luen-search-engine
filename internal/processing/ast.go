package processing

type NodeType int

const (
	TermNode NodeType = iota
	PhraseNode
	NotNode
	AndNode
	OrNode
)

type Node interface {
	typ NodeType
	// WIP
}

type Node interface {
	hasType() string
	Eval() ([]Result, error)
}

type TermNode struct {
	Token Token
}

type PhraseNode struct {
	Tokens []Token
}

type AndNode struct {
	Children []Node
}

type OrNode struct {
	Children []Node
}

type NotNode struct {
	Child Node
}
