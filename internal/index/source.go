package index

// PostingSource exposes posting lists for token lookup regardless of the backing store.
type PostingSource interface {
	Lookup(token Token) (*PostingList, error)
}
