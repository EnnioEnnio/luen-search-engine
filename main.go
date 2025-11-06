package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"luen-search-engine/internal/data"
	"luen-search-engine/internal/index"
	"luen-search-engine/internal/output"
	"luen-search-engine/internal/search"
	"luen-search-engine/internal/text"
)

func main() {
	dataPath := flag.String("data", "data/msmarco-docs.tsv", "Path to the MS MARCO TSV file")
	limit := flag.Int("limit", 1000, "Maximum number of documents to load (0 means all)")
	batchSize := flag.Int("batchSize", 1000, "Number of documents to process per batch")
	flag.Parse()

	if *limit < 0 {
		log.Fatalf("limit must be non-negative (0 means all), got %d", *limit)
	}

	if *batchSize <= 0 {
		log.Fatalf("batch size must be positive, got %d", *batchSize)
	}

	tokenizer := text.NewTokenizer()

	// Create the inverted index that will be built incrementally
	inverted := make(index.InvertedIndex)
	batchCount := 0

	// Process data in batches
	fmt.Printf("Loading and indexing data in batches of %d documents...\n", *batchSize)
	dataset, err := data.LoadInBatches(*dataPath, *batchSize, *limit, func(batch []data.Document) error {
		batchCount++
		fmt.Printf("Processing batch %d (%d documents)...\n", batchCount, len(batch))
		index.AddDocuments(inverted, batch, tokenizer)
		return nil
	})
	if err != nil {
		log.Fatalf("failed to load data: %v", err)
	}
	fmt.Printf("Data loaded with %d documents in %d batches.\n", dataset.Size(), batchCount)
	fmt.Printf("Inverted Index created with %d unique tokens.\n", inverted.TokenCount())

	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Your Search Engine is ready! Type a search term to use it, >exit to quit")

	for {
		fmt.Print("\nSearch: ")
		entry, err := reader.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				fmt.Println("\nExiting search engine. Goodbye!")
				return
			}
			log.Fatalf("failed to read search term: %v", err)
		}

		searchTerm := strings.TrimSpace(strings.ToLower(entry))
		if searchTerm == "" {
			fmt.Println("Please enter a valid search term.")
			continue
		}
		if searchTerm == ">exit" {
			fmt.Println("Exiting search engine. Goodbye!")
			return
		}

		results, total, err := search.Search(inverted, tokenizer, searchTerm)
		if err != nil {
			fmt.Printf("Error while searching: %v\n", err)
			continue
		}

		output.PrintResults(results, dataset, total)
	}
}
