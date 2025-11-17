package model

// DocID is the first column of the dataset
type DocID = uint32

// Token is a word, either in a query, or in a document text corpus (title/body)
type Token = string

// Match keeps track of how often a token occurred in a document.
type Match struct {
	Token     Token
	Frequency uint32
	Positions []uint32
}

// Result represents the aggregated scores for a matching document.
type Result struct {
	DocID              DocID
	TotalTermFrequency uint32 // used for ranking only
	Matches            []Match
}
