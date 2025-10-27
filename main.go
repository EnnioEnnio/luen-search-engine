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
	flag.Parse()

	tokenizer := text.NewTokenizer()

	dataset, err := data.Load(*dataPath, *limit)
	if err != nil {
		log.Fatalf("failed to load data: %v", err)
	}
	fmt.Printf("Data loaded with %d documents.\n", dataset.Size())

	inverted := index.Build(dataset.Documents, tokenizer)
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

		results, err := search.Search(inverted, tokenizer, searchTerm)
		if err != nil {
			fmt.Printf("Error while searching: %v\n", err)
			continue
		}

		total := len(results)
		output.PrintResults(results, dataset, total)
	}
}
