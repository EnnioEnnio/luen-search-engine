package model

// DocID is the first column of the dataset
type DocID = uint32

// Token is a word, either in a query, or in a document text corpus (title/body)
type Token = string

// Match keeps track of how often a token occurred in a document.
type Match struct {
	Token     Token
	Frequency int // Total frequency (title + body)
	TF_title  int // Frequency in title field
	TF_body   int // Frequency in body field
	Positions []int
}

// Result represents the aggregated scores for a matching document.
type Result struct {
	DocID              DocID
	TotalTermFrequency int // used for ranking only
	Matches            []Match
	Score              float64
}

// Document lengths stores the number of tokens in the title and body fields of a document.
type FieldDocLengths struct {
	TitleLength uint32
	BodyLength  uint32
}
