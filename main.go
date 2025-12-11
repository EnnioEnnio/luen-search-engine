package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"strings"
	"time"

	"luen-search-engine/internal/data"
	"luen-search-engine/internal/index"
	"luen-search-engine/internal/index/disk"
	"luen-search-engine/internal/indexer"
	"luen-search-engine/internal/output"
	"luen-search-engine/internal/search"
	"luen-search-engine/internal/synonyms"
	"luen-search-engine/internal/text"
)

// Profiler memory dump
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

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// Argparse and profiler setup
	dataPath := flag.String("data", "data/msmarco-docs-preprocessed.tsv", "Path to the MS MARCO TSV file")
	limit := flag.Int("limit", 10000, "Maximum number of documents to load (0 means all)")
	buildIndex := flag.Bool("buildindex", false, "Build the on-disk index and exit")
	diskLimit := flag.Int("disklimit", 100000, "Number of documents to index when automatically preparing the disk-backed serving layer")
	indexDir := flag.String("indexdir", "index", "Directory to store the on-disk index")
	batchBytes := flag.Int64("indexbatch", 64*1024*1024, "Approximate batch size in bytes for external indexing")
	diskCache := flag.Int("diskcache", 2048, "Number of posting lists to cache")
	cpuprofile := flag.String("cpuprofile", "", "Write CPU profile to file")
	memprofile := flag.String("memprofile", "", "Write memory profile to file")
	enableSynonyms := flag.Bool("enableSynonyms", false, "Enable synonym expansion using the SPLADE-like model")
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

	// Search engine setup
	tokenizer := text.NewTokenizer()

	if *buildIndex {
		builder := indexer.NewBuilder(indexer.Config{
			DataPath:     *dataPath,
			Limit:        *limit,
			BatchBytes:   *batchBytes,
			OutputDir:    *indexDir,
			KeepPartials: false,
			Tokenizer:    tokenizer,
		})
		builderStart := time.Now()
		manifest, err := builder.Build()
		if err != nil {
			log.Fatalf("failed to build disk index: %v", err)
		}
		builderTime := time.Since(builderStart).Round(time.Millisecond)
		fmt.Printf("On-disk index created at %s with %d documents and %d tokens in %s\n", *indexDir, manifest.DocumentCount, manifest.TokenCount, builderTime)
		return
	}

	var cleanup []func()

	loadStart := time.Now()
	if err := ensureDiskIndex(*indexDir, *dataPath, *diskLimit, tokenizer, *batchBytes); err != nil {
		log.Fatalf("failed to prepare on-disk index: %v", err)
	}
	dict, err := disk.LoadDictionary(indexer.DictionaryPath(*indexDir))
	if err != nil {
		log.Fatalf("failed to load dictionary: %v", err)
	}
	postingSource, err := disk.NewPostingStore(dict, indexer.PostingsPath(*indexDir), *diskCache)
	if err != nil {
		log.Fatalf("failed to open postings: %v", err)
	}
	cleanup = append(cleanup, func() {
		_ = postingSource.Close()
	})
	docLookup, err := data.OpenDocumentStore(*indexDir)
	if err != nil {
		log.Fatalf("failed to open document store: %v", err)
	}
	cleanup = append(cleanup, func() {
		_ = docLookup.Close()
	})
	docLengths, err := index.LoadDocLengths(*indexDir)
	if err != nil {
		log.Fatalf("failed to load document lengths: %v", err)
	}
	fmt.Printf("Loaded dictionary with %d tokens from %s in %s.\n", dict.Size(), *indexDir, time.Since(loadStart).Round(time.Millisecond))

	for i := len(cleanup) - 1; i >= 0; i-- {
		defer cleanup[i]()
	}

	var synonymExpander *synonyms.SpladeLike

	if *enableSynonyms {
		se, err := synonyms.NewSpladeLike()
		if err != nil {
			log.Fatalf("failed to create synonym expander: %v", err)
		}
		synonymExpander = se
		fmt.Println("Synonym Expander model loaded.")
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Your Search Engine is ready! Type a search term to use it, >exit to quit")

	// Search loop
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
		var (
			results []search.Result
			total   int
		)
		if *enableSynonyms && synonymExpander != nil {
			results, total, err = search.Search(ctx, postingSource, tokenizer, searchTerm, docLengths, synonymExpander)
		} else {
			results, total, err = search.Search(ctx, postingSource, tokenizer, searchTerm, docLengths)
		}
		searchTime := time.Since(searchStart)
		if err != nil {
			fmt.Printf("Error while searching: %v\n", err)
			continue
		}

		output.PrintResults(results, docLookup, total, searchTime, docLengths)
	}
}

func ensureDiskIndex(dir, dataPath string, limit int, tokenizer *text.Tokenizer, batchBytes int64) error {
	if diskIndexExists(dir) {
		return nil
	}
	log.Printf("on-disk index missing at %s, building (limit=%d)...", dir, limit)
	builder := indexer.NewBuilder(indexer.Config{
		DataPath:   dataPath,
		Limit:      limit,
		BatchBytes: batchBytes,
		OutputDir:  dir,
		Tokenizer:  tokenizer,
	})
	_, err := builder.Build()
	return err
}

func diskIndexExists(dir string) bool {
	if _, err := os.Stat(indexer.DictionaryPath(dir)); err != nil {
		return false
	}
	if _, err := os.Stat(indexer.PostingsPath(dir)); err != nil {
		return false
	}
	if _, err := os.Stat(data.DocStoreIndexPath(dir)); err != nil {
		return false
	}
	if _, err := os.Stat(data.DocStoreDataPath(dir)); err != nil {
		return false
	}
	return true
}
