package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
	"strings"
	"time"

	"luen-search-engine/internal/data"
	"luen-search-engine/internal/index"
	"luen-search-engine/internal/output"
	"luen-search-engine/internal/search"
	"luen-search-engine/internal/text"
)

func main() {
	dataPath := flag.String("data", "data/msmarco-docs-preprocessed.tsv", "Path to the MS MARCO TSV file")
	limit := flag.Int("limit", 1000, "Maximum number of documents to load (0 means all)")
	mode := flag.String("mode", "single", "Choose query mode between single and phrase (for phrase queries). Default is single mode.")
	cpuprofile := flag.String("cpuprofile", "", "Write CPU profile to file")
	memprofile := flag.String("memprofile", "", "Write memory profile to file")

	flag.Parse()

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatalf("could not create CPU profile: %v", err)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			f.Close()
			log.Fatalf("could not start CPU profile: %v", err)
		}
		defer func() {
			pprof.StopCPUProfile()
			f.Close()
		}()
	}

	if *memprofile != "" {
		defer writeMemProfile(*memprofile)
	}

	if *mode != "single" && *mode != "phrase" {
		log.Fatalf("invalid mode: %s", *mode)
	}
	fmt.Printf("Using mode: %s\n", *mode)

	tokenizer := text.NewTokenizer()

	loadStart := time.Now()
	dataset, err := data.Load(*dataPath, *limit)
	if err != nil {
		log.Fatalf("failed to load data: %v", err)
	}
	fmt.Printf("Data loaded with %d documents in %s.\n", dataset.Size(), time.Since(loadStart).Round(time.Millisecond))

	indexStart := time.Now()
	inverted := index.Build(dataset.Documents, tokenizer)
	fmt.Printf("Inverted Index created with %d unique tokens in %s.\n", inverted.TokenCount(), time.Since(indexStart).Round(time.Millisecond))

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

		searchStart := time.Now()
		results, total, err := search.Search(inverted, tokenizer, searchTerm, *mode)
		searchTime := time.Since(searchStart)
		if err != nil {
			fmt.Printf("Error while searching: %v\n", err)
			continue
		}

		output.PrintResults(results, dataset, total, searchTime)
	}
}

func writeMemProfile(path string) {
	if path == "" {
		return
	}

	f, err := os.Create(path)
	if err != nil {
		log.Printf("could not create memory profile: %v", err)
		return
	}
	defer f.Close()

	runtime.GC()
	if err := pprof.WriteHeapProfile(f); err != nil {
		log.Printf("could not write memory profile: %v", err)
	}
}
