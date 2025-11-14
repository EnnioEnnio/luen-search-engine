package benchutil

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"luen-search-engine/internal/data"
	"luen-search-engine/internal/text"
)

const (
	DatasetPath   = "data/msmarco-docs-bench-100000.tsv"
	QueriesPath   = "data/msmarco-queries-bench-1000.txt"
	DatasetLimit  = 100000
	QueryCount    = 1000
	minQueryTerms = 1
	maxQueryTerms = 5
)

var (
	datasetAbsPathOnce sync.Once
	datasetOnce        sync.Once
	dataset            *data.Dataset
	datasetErr         error
	datasetAbsPath     string
	datasetPathErr     error

	queriesAbsPathOnce sync.Once
	queriesOnce        sync.Once
	cachedQueries      []string
	queriesErr         error
	queriesAbsPath     string
	queriesPathErr     error
)

// LoadDataset loads the benchmark dataset once and caches it for reuse.
func LoadDataset(tb testing.TB) *data.Dataset {
	tb.Helper()
	path := benchmarkDatasetPath(tb)
	datasetOnce.Do(func() {
		dataset, datasetErr = data.Load(path, DatasetLimit)
	})
	if datasetErr != nil {
		tb.Fatalf("bench dataset error (%s): %v", path, datasetErr)
	}
	if dataset == nil || len(dataset.Documents) == 0 {
		tb.Fatalf("bench dataset is empty: %s", DatasetPath)
	}
	return dataset
}

// BuildQueries builds a deterministic query slice without relying on testing helpers.
func BuildQueries(docs []data.Document, tokenizer *text.Tokenizer, count int) ([]string, error) {
	if len(docs) == 0 {
		return nil, fmt.Errorf("cannot generate queries without documents")
	}
	if count <= 0 {
		return nil, fmt.Errorf("query count must be positive")
	}
	return generateQueries(docs, tokenizer, count), nil
}

// GenerateQueries is a testing helper wrapper around BuildQueries.
func GenerateQueries(tb testing.TB, docs []data.Document, tokenizer *text.Tokenizer, count int) []string {
	tb.Helper()
	queries, err := BuildQueries(docs, tokenizer, count)
	if err != nil {
		tb.Fatalf("generate queries: %v", err)
	}
	return queries
}

// LoadQueries loads the cached benchmark queries from disk.
func LoadQueries(tb testing.TB) []string {
	tb.Helper()
	path := benchmarkQueriesPath(tb)
	queriesOnce.Do(func() {
		data, err := os.ReadFile(path)
		if err != nil {
			queriesErr = fmt.Errorf("read queries file %s: %w", path, err)
			return
		}
		lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
		out := make([]string, 0, len(lines))
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			out = append(out, line)
		}
		if len(out) == 0 {
			queriesErr = fmt.Errorf("query file %s is empty", path)
			return
		}
		cachedQueries = out
	})

	if queriesErr != nil {
		tb.Fatalf("bench queries error: %v", queriesErr)
	}
	return cachedQueries
}

func benchmarkDatasetPath(tb testing.TB) string {
	tb.Helper()
	datasetAbsPathOnce.Do(func() {
		root, err := findRepoRoot()
		if err != nil {
			datasetPathErr = err
			return
		}
		datasetAbsPath = filepath.Join(root, DatasetPath)
	})
	if datasetPathErr != nil {
		tb.Fatalf("bench dataset path error: %v", datasetPathErr)
	}
	return datasetAbsPath
}

func benchmarkQueriesPath(tb testing.TB) string {
	tb.Helper()
	queriesAbsPathOnce.Do(func() {
		root, err := findRepoRoot()
		if err != nil {
			queriesPathErr = err
			return
		}
		queriesAbsPath = filepath.Join(root, QueriesPath)
	})
	if queriesPathErr != nil {
		tb.Fatalf("bench queries path error: %v", queriesPathErr)
	}
	return queriesAbsPath
}

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found when resolving benchmark dataset path")
		}
		dir = parent
	}
}

func generateQueries(docs []data.Document, tokenizer *text.Tokenizer, count int) []string {
	queries := make([]string, 0, count)
	rng := rand.New(rand.NewSource(42))

	for len(queries) < count {
		doc := docs[rng.Intn(len(docs))]
		tokens := tokenizer.Tokenize(doc.Text)
		if len(tokens) == 0 {
			continue
		}

		maxLen := maxQueryTerms
		if len(tokens) < maxLen {
			maxLen = len(tokens)
		}
		length := rng.Intn(maxLen-minQueryTerms+1) + minQueryTerms
		if length > len(tokens) {
			length = len(tokens)
		}

		startMax := len(tokens) - length
		start := 0
		if startMax > 0 {
			start = rng.Intn(startMax + 1)
		}
		chunk := tokens[start : start+length]

		var builder strings.Builder
		builder.Grow(length * 10)
		builder.WriteString(chunk[0])

		includeOR := length > 1 && rng.Intn(5) == 0
		for i := 1; i < len(chunk); i++ {
			if includeOR && i == 1 {
				builder.WriteString(" or ")
			} else {
				builder.WriteByte(' ')
			}
			builder.WriteString(chunk[i])
		}

		queries = append(queries, builder.String())
	}

	return queries
}
