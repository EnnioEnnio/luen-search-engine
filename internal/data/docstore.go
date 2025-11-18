package data

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

const (
	docStoreDataFile  = "docs.bin"
	docStoreIndexFile = "docs.idx"
)

// DocStoreDataPath returns the fully-qualified path to docs.bin inside dir.
func DocStoreDataPath(dir string) string {
	return filepath.Join(dir, docStoreDataFile)
}

// DocStoreIndexPath returns the fully-qualified path to docs.idx inside dir.
func DocStoreIndexPath(dir string) string {
	return filepath.Join(dir, docStoreIndexFile)
}

type docIndexEntry struct {
	offset int64
	length uint32
}

// DocumentStoreWriter persists dataset records to disk for random access during serving.
type DocumentStoreWriter struct {
	dataFile  *os.File
	indexFile *os.File
	writer    *bufio.Writer
	indexW    *bufio.Writer
	offset    int64
}

// NewDocumentStoreWriter creates the docstore files beneath dir (typically the index dir).
func NewDocumentStoreWriter(dir string) (*DocumentStoreWriter, error) {
	dataPath := filepath.Join(dir, docStoreDataFile)
	idxPath := filepath.Join(dir, docStoreIndexFile)
	dataFile, err := os.Create(dataPath)
	if err != nil {
		return nil, fmt.Errorf("create docstore data: %w", err)
	}
	indexFile, err := os.Create(idxPath)
	if err != nil {
		dataFile.Close()
		return nil, fmt.Errorf("create docstore index: %w", err)
	}
	return &DocumentStoreWriter{
		dataFile:  dataFile,
		indexFile: indexFile,
		writer:    bufio.NewWriter(dataFile),
		indexW:    bufio.NewWriter(indexFile),
	}, nil
}

// Append writes the next document record and updates the doc index.
func (w *DocumentStoreWriter) Append(doc Document) error {
	if w == nil {
		return fmt.Errorf("docstore writer is nil")
	}
	start := w.offset
	bytesWritten := int64(0)
	if err := binary.Write(w.writer, binary.LittleEndian, doc.ID); err != nil {
		return fmt.Errorf("write doc id: %w", err)
	}
	bytesWritten += 4
	if n, err := writeString(w.writer, doc.URL); err != nil {
		return err
	} else {
		bytesWritten += n
	}
	if n, err := writeString(w.writer, doc.Title); err != nil {
		return err
	} else {
		bytesWritten += n
	}
	if n, err := writeString(w.writer, doc.Text); err != nil {
		return err
	} else {
		bytesWritten += n
	}
	w.offset += bytesWritten
	if err := binary.Write(w.indexW, binary.LittleEndian, doc.ID); err != nil {
		return fmt.Errorf("write index doc id: %w", err)
	}
	if err := binary.Write(w.indexW, binary.LittleEndian, start); err != nil {
		return fmt.Errorf("write index offset: %w", err)
	}
	if err := binary.Write(w.indexW, binary.LittleEndian, uint32(bytesWritten)); err != nil {
		return fmt.Errorf("write index length: %w", err)
	}
	return nil
}

// Close flushes and closes the underlying files.
func (w *DocumentStoreWriter) Close() error {
	if w == nil {
		return nil
	}
	if err := w.writer.Flush(); err != nil {
		return err
	}
	if err := w.indexW.Flush(); err != nil {
		return err
	}
	if err := w.dataFile.Close(); err != nil {
		return err
	}
	return w.indexFile.Close()
}

func writeString(w *bufio.Writer, value string) (int64, error) {
	if len(value) > int(^uint32(0)) {
		return 0, fmt.Errorf("string too long")
	}
	if err := binary.Write(w, binary.LittleEndian, uint32(len(value))); err != nil {
		return 0, err
	}
	if _, err := w.WriteString(value); err != nil {
		return 0, err
	}
	return int64(4 + len(value)), nil
}

// DocumentStore provides read-only random access to documents stored on disk.
type DocumentStore struct {
	data  *os.File
	index map[uint32]docIndexEntry
	mu    sync.Mutex
}

var _ DocumentLookup = (*DocumentStore)(nil)

// OpenDocumentStore loads the on-disk index metadata and prepares for lookups.
func OpenDocumentStore(dir string) (*DocumentStore, error) {
	idxPath := filepath.Join(dir, docStoreIndexFile)
	dataPath := filepath.Join(dir, docStoreDataFile)
	indexFile, err := os.Open(idxPath)
	if err != nil {
		return nil, fmt.Errorf("open docstore index: %w", err)
	}
	defer indexFile.Close()

	entries := make(map[uint32]docIndexEntry)
	reader := bufio.NewReader(indexFile)
	for {
		var docID uint32
		if err := binary.Read(reader, binary.LittleEndian, &docID); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("read docstore index doc id: %w", err)
		}
		var offset int64
		if err := binary.Read(reader, binary.LittleEndian, &offset); err != nil {
			return nil, fmt.Errorf("read docstore index offset: %w", err)
		}
		var length uint32
		if err := binary.Read(reader, binary.LittleEndian, &length); err != nil {
			return nil, fmt.Errorf("read docstore index length: %w", err)
		}
		entries[docID] = docIndexEntry{offset: offset, length: length}
	}

	dataFile, err := os.Open(dataPath)
	if err != nil {
		return nil, fmt.Errorf("open docstore data: %w", err)
	}

	return &DocumentStore{data: dataFile, index: entries}, nil
}

// Close releases the open file descriptor.
func (s *DocumentStore) Close() error {
	if s == nil || s.data == nil {
		return nil
	}
	return s.data.Close()
}

// Lookup returns the requested document by seeking into the docstore file.
func (s *DocumentStore) Lookup(docID uint32) (Document, bool) {
	if s == nil {
		return Document{}, false
	}
	entry, ok := s.index[docID]
	if !ok {
		return Document{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.data.Seek(entry.offset, io.SeekStart); err != nil {
		return Document{}, false
	}
	buf := make([]byte, entry.length)
	if _, err := io.ReadFull(s.data, buf); err != nil {
		return Document{}, false
	}
	reader := bytesReader{buf: buf}
	return reader.readDocument()
}

type bytesReader struct {
	buf []byte
	pos int
}

func (r *bytesReader) readDocument() (Document, bool) {
	var doc Document
	if !r.readUint32(&doc.ID) {
		return Document{}, false
	}
	url, ok := r.readString()
	if !ok {
		return Document{}, false
	}
	title, ok := r.readString()
	if !ok {
		return Document{}, false
	}
	text, ok := r.readString()
	if !ok {
		return Document{}, false
	}
	doc.URL = url
	doc.Title = title
	doc.Text = text
	return doc, true
}

func (r *bytesReader) readUint32(out *uint32) bool {
	if r.pos+4 > len(r.buf) {
		return false
	}
	*out = binary.LittleEndian.Uint32(r.buf[r.pos : r.pos+4])
	r.pos += 4
	return true
}

func (r *bytesReader) readString() (string, bool) {
	var length uint32
	if !r.readUint32(&length) {
		return "", false
	}
	end := r.pos + int(length)
	if end > len(r.buf) {
		return "", false
	}
	value := string(r.buf[r.pos:end])
	r.pos = end
	return value, true
}
