package indexer

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"luen-search-engine/internal/data"
	"luen-search-engine/internal/index"
	"luen-search-engine/internal/text"
)

const (
	defaultBatchBytes  = 64 * 1024 * 1024
	partialFilePattern = "partial-%06d.gob"
	postingsFileName   = "postings.bin"
	dictionaryFileName = "dictionary.tsv"
	manifestFileName   = "manifest.json"
)

// Config drives how the external indexer batches and persists data.
type Config struct {
	DataPath     string
	Limit        int
	BatchBytes   int64
	OutputDir    string
	TempDir      string
	KeepPartials bool
	Tokenizer    *text.Tokenizer
}

// Builder orchestrates the SPIMI/blocked indexing flow.
type Builder struct {
	cfg Config
}

// Manifest captures the resulting on-disk index metadata.
type Manifest struct {
	DocumentCount int       `json:"documentCount"`
	TokenCount    int       `json:"tokenCount"`
	PartialFiles  int       `json:"partialFiles"`
	BatchBytes    int64     `json:"batchBytes"`
	CreatedAt     time.Time `json:"createdAt"`
}

// NewBuilder applies defaults and returns a configured Builder.
func NewBuilder(cfg Config) *Builder {
	if cfg.BatchBytes <= 0 {
		cfg.BatchBytes = defaultBatchBytes
	}
	return &Builder{cfg: cfg}
}

// Build runs the external indexing pipeline and writes the final on-disk index.
func (b *Builder) Build() (*Manifest, error) {
	if err := b.validate(); err != nil {
		return nil, err
	}

	if err := os.MkdirAll(b.cfg.OutputDir, 0o755); err != nil {
		return nil, fmt.Errorf("create index dir: %w", err)
	}

	tempDir := b.cfg.TempDir
	if tempDir == "" {
		tempDir = filepath.Join(b.cfg.OutputDir, "tmp")
	}
	if err := os.MkdirAll(tempDir, 0o755); err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}

	loader, err := data.NewBatchLoader(b.cfg.DataPath, b.cfg.Limit, b.cfg.BatchBytes)
	if err != nil {
		return nil, fmt.Errorf("create batch loader: %w", err)
	}
	defer loader.Close()

	var partialPaths []string
	totalProcessed := 0
	for batch := 0; ; batch++ {
		docs, err := loader.NextBatch()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read batch: %w", err)
		}
		if len(docs) == 0 {
			continue
		}
		totalProcessed = totalProcessed + len(docs)
		log.Printf("indexing batch %d (%d docs --- %d/3213835 in total)", batch+1, len(docs), totalProcessed)
		partial := index.Build(docs, b.cfg.Tokenizer)
		partialPath := filepath.Join(tempDir, fmt.Sprintf(partialFilePattern, batch))
		if err := spillPartialIndex(partialPath, partial); err != nil {
			return nil, fmt.Errorf("write partial index: %w", err)
		}
		partialPaths = append(partialPaths, partialPath)

		for i := range docs {
			docs[i] = data.Document{}
		}
	}

	if len(partialPaths) == 0 {
		return nil, fmt.Errorf("no documents indexed from %s", b.cfg.DataPath)
	}

	log.Printf("merging %d partial indexes", len(partialPaths))
	tokenCount, err := mergePartials(partialPaths, b.cfg.OutputDir, b.cfg.KeepPartials)
	if err != nil {
		return nil, err
	}

	manifest := &Manifest{
		DocumentCount: loader.DocsRead(),
		TokenCount:    tokenCount,
		PartialFiles:  len(partialPaths),
		BatchBytes:    b.cfg.BatchBytes,
		CreatedAt:     time.Now().UTC(),
	}

	if err := writeManifest(filepath.Join(b.cfg.OutputDir, manifestFileName), manifest); err != nil {
		return nil, err
	}

	return manifest, nil
}

// validate ensures the builder configuration contains sane values before indexing begins.
func (b *Builder) validate() error {
	if b.cfg.DataPath == "" {
		return errors.New("data path must be provided")
	}
	if b.cfg.OutputDir == "" {
		return errors.New("output directory must be provided")
	}
	if b.cfg.Tokenizer == nil {
		return errors.New("tokenizer must not be nil")
	}
	return nil
}

func writeManifest(path string, manifest *Manifest) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create manifest: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(manifest); err != nil {
		return fmt.Errorf("encode manifest: %w", err)
	}
	return nil
}
