package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"luen-search-engine/internal/benchutil"
	"luen-search-engine/internal/data"
	"luen-search-engine/internal/text"
)

func main() {
	datasetPath := flag.String("dataset", benchutil.DatasetPath, "Path to benchmark dataset")
	outputPath := flag.String("output", benchutil.QueriesPath, "Where to write the generated queries")
	limit := flag.Int("limit", benchutil.DatasetLimit, "Number of documents to load from the dataset (0 = all)")
	count := flag.Int("count", benchutil.QueryCount, "Number of queries to generate")
	flag.Parse()

	ds, err := data.Load(*datasetPath, *limit)
	if err != nil {
		log.Fatalf("load dataset: %v", err)
	}
	if ds.Size() == 0 {
		log.Fatalf("dataset %s returned no documents", *datasetPath)
	}

	tokenizer := text.NewTokenizer()
	queries, err := benchutil.BuildQueries(ds.Documents, tokenizer, *count)
	if err != nil {
		log.Fatalf("generate queries: %v", err)
	}

	if err := writeQueries(*outputPath, queries); err != nil {
		log.Fatalf("write queries: %v", err)
	}
	fmt.Printf("Generated %d queries at %s\n", len(queries), *outputPath)
}

func writeQueries(path string, queries []string) error {
	content := strings.Join(queries, "\n") + "\n"
	return os.WriteFile(path, []byte(content), 0o600)
}
