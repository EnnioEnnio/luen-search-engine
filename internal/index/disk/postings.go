package disk

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sync"

	"container/list"

	"luen-search-engine/internal/index"
)

// PostingStore lazily reads posting lists from postings.bin and caches hot entries.
type PostingStore struct {
	dict  *Dictionary
	file  *os.File
	cache *lruCache
	mu    sync.Mutex
}

// NewPostingStore opens postings.bin and prepares a cache of the given size.
func NewPostingStore(dict *Dictionary, path string, cacheSize int) (*PostingStore, error) {
	if dict == nil {
		return nil, fmt.Errorf("dictionary must not be nil")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open postings: %w", err)
	}
	if cacheSize <= 0 {
		cacheSize = 1024
	}
	return &PostingStore{
		dict:  dict,
		file:  file,
		cache: newLRUCache(cacheSize),
	}, nil
}

// Close releases the file handle underlying the posting store.
func (p *PostingStore) Close() error {
	if p == nil || p.file == nil {
		return nil
	}
	return p.file.Close()
}

// Lookup loads a posting list for the specified token, optionally reusing a cached copy.
func (p *PostingStore) Lookup(token index.Token) (*index.PostingList, error) {
	if p == nil {
		return nil, nil
	}
	if posting, ok := p.cache.Get(string(token)); ok {
		return posting, nil
	}
	entry, ok := p.dict.Lookup(string(token))
	if !ok {
		return nil, nil
	}
	data := make([]byte, entry.Length)
	p.mu.Lock()
	if _, err := p.file.Seek(entry.Offset, io.SeekStart); err != nil {
		p.mu.Unlock()
		return nil, fmt.Errorf("seek postings for %s: %w", token, err)
	}
	if _, err := io.ReadFull(p.file, data); err != nil {
		p.mu.Unlock()
		return nil, fmt.Errorf("read postings for %s: %w", token, err)
	}
	p.mu.Unlock()

	posting, err := decodePosting(data)
	if err != nil {
		return nil, fmt.Errorf("decode postings for %s: %w", token, err)
	}
	p.cache.Add(string(token), posting)
	return posting, nil
}

func decodePosting(data []byte) (*index.PostingList, error) {
	reader := bytes.NewReader(data)
	var docCount uint32
	if err := binary.Read(reader, binary.LittleEndian, &docCount); err != nil {
		return nil, err
	}
	posting := &index.PostingList{
		DocFreq: int(docCount),
		Docs:    make(map[index.DocID][]index.Position, docCount),
	}
	for i := uint32(0); i < docCount; i++ {
		var docID uint32
		if err := binary.Read(reader, binary.LittleEndian, &docID); err != nil {
			return nil, err
		}
		var posCount uint32
		if err := binary.Read(reader, binary.LittleEndian, &posCount); err != nil {
			return nil, err
		}
		positions := make([]index.Position, posCount)
		for j := uint32(0); j < posCount; j++ {
			var pos uint32
			if err := binary.Read(reader, binary.LittleEndian, &pos); err != nil {
				return nil, err
			}
			positions[j] = index.Position(pos)
		}
		posting.Docs[index.DocID(docID)] = positions
	}
	return posting, nil
}

type lruCache struct {
	capacity int
	entries  map[string]*list.Element
	ll       *list.List
	mu       sync.Mutex
}

type kv struct {
	key   string
	value *index.PostingList
}

func newLRUCache(capacity int) *lruCache {
	return &lruCache{
		capacity: capacity,
		entries:  make(map[string]*list.Element, capacity),
		ll:       list.New(),
	}
}

func (c *lruCache) Get(key string) (*index.PostingList, bool) {
	if c == nil {
		return nil, false
	}
	c.mu.Lock()
	if elem, ok := c.entries[key]; ok {
		c.ll.MoveToFront(elem)
		c.mu.Unlock()
		return elem.Value.(kv).value, true
	}
	c.mu.Unlock()
	return nil, false
}

func (c *lruCache) Add(key string, value *index.PostingList) {
	if c == nil {
		return
	}
	c.mu.Lock()
	if elem, ok := c.entries[key]; ok {
		c.ll.MoveToFront(elem)
		c.mu.Unlock()
		elem.Value = kv{key: key, value: value}
		return
	}
	elem := c.ll.PushFront(kv{key: key, value: value})
	c.entries[key] = elem
	if c.ll.Len() > c.capacity {
		c.evict()
	}
	c.mu.Unlock()
}

func (c *lruCache) evict() {
	if c == nil {
		return
	}
	back := c.ll.Back()
	if back == nil {
		return
	}
	c.ll.Remove(back)
	entry := back.Value.(kv)
	delete(c.entries, entry.key)
}
