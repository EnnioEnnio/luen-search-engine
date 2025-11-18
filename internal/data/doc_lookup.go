package data

// DocumentLookup provides random access to documents by ID without exposing the underlying storage.
type DocumentLookup interface {
	Lookup(docID uint32) (Document, bool)
}

// Lookup implements DocumentLookup for in-memory datasets.
func (d *Dataset) Lookup(docID uint32) (Document, bool) {
	if d == nil || d.ByID == nil {
		return Document{}, false
	}
	doc, ok := d.ByID[docID]
	return doc, ok
}
