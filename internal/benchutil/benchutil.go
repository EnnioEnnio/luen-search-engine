package benchutil

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
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

// DatasetPathAbs returns the absolute filesystem path to the benchmark dataset TSV.
func DatasetPathAbs(tb testing.TB) string {
	return benchmarkDatasetPath(tb)
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

// QueriesPathAbs returns the absolute filesystem path to the benchmark queries file.
func QueriesPathAbs(tb testing.TB) string {
	return benchmarkQueriesPath(tb)
}

// WithMutedLogs temporarily silences the global logger while executing fn.
func WithMutedLogs(tb testing.TB, fn func()) {
	tb.Helper()
	prev := log.Writer()
	log.SetOutput(io.Discard)
	defer log.SetOutput(prev)
	fn()
}

// LogBenchmarkSummary renders a consistent summary for benchmark metrics.
func LogBenchmarkSummary(tb testing.TB, name string, fields map[string]interface{}) {
	tb.Helper()
	if len(fields) == 0 {
		return
	}
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	tb.Logf("=== %s ===", name)
	for _, key := range keys {
		tb.Logf("  %-20s %v", key+":", fields[key])
	}
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

	patterns := []func(*rand.Rand, []string) (string, bool){
		singleTermQuery,
		andQuery,
		orChainQuery,
		notGroupQuery,
		notGroupOrTermQuery,
		phraseOnlyQuery,
		phraseWithTermsQuery,
		mixedPhraseQuery,
		groupedPhraseQuery,
		longMixedQuery,
	}

	for len(queries) < count {
		doc := docs[rng.Intn(len(docs))]
		tokens := tokenizer.Tokenize(doc.Text)
		if len(tokens) == 0 {
			continue
		}

		pattern := patterns[rng.Intn(len(patterns))]
		if query, ok := pattern(rng, tokens); ok {
			queries = append(queries, query)
		}
	}

	return queries
}

func singleTermQuery(rng *rand.Rand, tokens []string) (string, bool) {
	terms, ok := randomUniqueTerms(rng, tokens, 1)
	if !ok {
		return "", false
	}
	return terms[0], true
}

func andQuery(rng *rand.Rand, tokens []string) (string, bool) {
	terms, ok := randomUniqueTerms(rng, tokens, 2)
	if !ok {
		return "", false
	}
	return fmt.Sprintf("%s and %s", terms[0], terms[1]), true
}

func orChainQuery(rng *rand.Rand, tokens []string) (string, bool) {
	length := 2 + rng.Intn(2) // 2-3 terms
	terms, ok := randomUniqueTerms(rng, tokens, length)
	if !ok {
		return "", false
	}
	return strings.Join(terms, " or "), true
}

func notGroupQuery(rng *rand.Rand, tokens []string) (string, bool) {
	terms, ok := randomUniqueTerms(rng, tokens, 2)
	if !ok {
		return "", false
	}
	return fmt.Sprintf("(%s and not %s)", terms[0], terms[1]), true
}

func notGroupOrTermQuery(rng *rand.Rand, tokens []string) (string, bool) {
	terms, ok := randomUniqueTerms(rng, tokens, 3)
	if !ok {
		return "", false
	}
	return fmt.Sprintf("(%s and not %s) or %s", terms[0], terms[1], terms[2]), true
}

func phraseOnlyQuery(rng *rand.Rand, tokens []string) (string, bool) {
	phrase, ok := randomPhrase(rng, tokens, 5)
	if !ok {
		return "", false
	}
	return fmt.Sprintf("\"%s\"", phrase), true
}

func phraseWithTermsQuery(rng *rand.Rand, tokens []string) (string, bool) {
	phrase, ok := randomPhrase(rng, tokens, 5)
	if !ok {
		return "", false
	}
	terms, ok := randomUniqueTerms(rng, tokens, 2)
	if !ok {
		return "", false
	}

	if rng.Intn(2) == 0 {
		return fmt.Sprintf("\"%s\" and %s", phrase, terms[0]), true
	}
	return fmt.Sprintf("%s and \"%s\" and %s", terms[0], phrase, terms[1]), true
}

func mixedPhraseQuery(rng *rand.Rand, tokens []string) (string, bool) {
	phrase, ok := randomPhrase(rng, tokens, 4)
	if !ok {
		return "", false
	}
	terms, ok := randomUniqueTerms(rng, tokens, 2)
	if !ok {
		return "", false
	}

	return fmt.Sprintf("%s and \"%s\" or %s", terms[0], phrase, terms[1]), true
}

func groupedPhraseQuery(rng *rand.Rand, tokens []string) (string, bool) {
	phrase, ok := randomPhrase(rng, tokens, 4)
	if !ok {
		return "", false
	}
	terms, ok := randomUniqueTerms(rng, tokens, 2)
	if !ok {
		return "", false
	}
	return fmt.Sprintf("%s and (%s or \"%s\")", terms[0], terms[1], phrase), true
}

func longMixedQuery(rng *rand.Rand, tokens []string) (string, bool) {
	terms, ok := randomUniqueTerms(rng, tokens, 4)
	if !ok {
		return "", false
	}
	phrase, ok := randomPhrase(rng, tokens, 6)
	if !ok {
		return "", false
	}
	return fmt.Sprintf("%s and %s or (%s and not %s) and \"%s\"", terms[0], terms[1], terms[2], terms[3], phrase), true
}

func randomUniqueTerms(rng *rand.Rand, tokens []string, count int) ([]string, bool) {
	if len(tokens) < count {
		return nil, false
	}

	unique := make([]string, 0, len(tokens))
	seen := make(map[string]struct{}, len(tokens))
	for _, token := range tokens {
		if token == "" {
			continue
		}
		if _, ok := seen[token]; ok {
			continue
		}
		seen[token] = struct{}{}
		unique = append(unique, token)
	}

	if len(unique) < count {
		return nil, false
	}

	rng.Shuffle(len(unique), func(i, j int) {
		unique[i], unique[j] = unique[j], unique[i]
	})

	return unique[:count], true
}

func randomPhrase(rng *rand.Rand, tokens []string, maxWords int) (string, bool) {
	if len(tokens) < 2 {
		return "", false
	}
	if maxWords < 2 {
		maxWords = 2
	}
	if maxWords > len(tokens) {
		maxWords = len(tokens)
	}

	length := 2
	if maxWords > 2 {
		length += rng.Intn(maxWords - 1)
	}

	if length > len(tokens) {
		length = len(tokens)
	}

	startMax := len(tokens) - length
	start := 0
	if startMax > 0 {
		start = rng.Intn(startMax + 1)
	}

	phrase := strings.TrimSpace(strings.Join(tokens[start:start+length], " "))
	if phrase == "" {
		return "", false
	}
	return phrase, true
}
