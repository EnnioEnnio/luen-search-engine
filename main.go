package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"strings"
	"time"

	"luen-search-engine/internal/ai"
	"luen-search-engine/internal/data"
	"luen-search-engine/internal/index"
	"luen-search-engine/internal/index/disk"
	"luen-search-engine/internal/indexer"
	"luen-search-engine/internal/output"
	"luen-search-engine/internal/pb"
	"luen-search-engine/internal/search"
	"luen-search-engine/internal/synonyms"
	"luen-search-engine/internal/text"

	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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
	if err := godotenv.Load(); err != nil {
		log.Printf("No .env file found or error loading it: %v", err)
	}

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
	// Server mode
	serverMode := flag.Bool("server", false, "Start HTTP server")
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

	// gRPC Client setup for semantic search
	var semanticClient *search.SemanticClient
	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Printf("Failed to create semantic search client: %v. Semantic search will be disabled.", err)
	} else {
		defer conn.Close()
		semanticClient = &search.SemanticClient{
			Client: pb.NewSemanticEmbeddingServiceClient(conn),
		}
	}

	if *serverMode {
		var aiClient ai.Client
		rawAPIKey := os.Getenv("OPENAI_API_KEY")
		apiKey := strings.TrimSpace(rawAPIKey)
		if apiKey != "" {
			aiClient = ai.NewOpenAIClient(apiKey)
			log.Println("OpenAI client initialized")
		} else {
			log.Println("OPENAI_API_KEY not set, AI answers will be disabled")
		}

		startServer(ctx, postingSource, docLookup, tokenizer, docLengths, semanticClient, aiClient)
		return
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
			semanticResults []search.Result
			bm25Results     []search.Result
			total           int
		)
		if *enableSynonyms && synonymExpander != nil {
			semanticResults, bm25Results, total, err = search.Search(ctx, postingSource, tokenizer, semanticClient, searchTerm, docLengths, synonymExpander)
		} else {
			semanticResults, bm25Results, total, err = search.Search(ctx, postingSource, tokenizer, semanticClient, searchTerm, docLengths)
		}
		searchTime := time.Since(searchStart)
		if err != nil {
			fmt.Printf("Error while searching: %v\n", err)
			continue
		}

		output.PrintResults(semanticResults, bm25Results, docLookup, total, searchTime, docLengths)
	}
}

func startServer(ctx context.Context, postingSource *disk.PostingStore, docLookup *data.DocumentStore, tokenizer *text.Tokenizer, docLengths map[search.DocID]search.FieldDocLength, semanticClient *search.SemanticClient, aiClient ai.Client) {
	http.Handle("/", http.FileServer(http.Dir("./static")))

	http.HandleFunc("/api/search", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("q")
		if query == "" {
			http.Error(w, "Missing query parameter 'q'", http.StatusBadRequest)
			return
		}

		start := time.Now()
		semanticResults, bm25Results, total, err := search.Search(r.Context(), postingSource, tokenizer, semanticClient, strings.ToLower(query), docLengths)
		duration := time.Since(start)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		type Result struct {
			ID      string  `json:"id"`
			Title   string  `json:"title"`
			URL     string  `json:"url"`
			Content string  `json:"content"`
			Score   float64 `json:"score"`
		}

		type Response struct {
			SemanticResults []Result `json:"semantic_results"`
			BM25Results     []Result `json:"bm25_results"`
			Total           int      `json:"total"`
			Duration        string   `json:"duration"`
		}

		var jsonSemanticResults []Result
		for _, res := range semanticResults {
			doc, ok := docLookup.Lookup(res.DocID)
			if !ok {
				continue
			}
			jsonSemanticResults = append(jsonSemanticResults, Result{
				ID:      fmt.Sprint(res.DocID),
				Title:   doc.Title,
				URL:     doc.URL,
				Content: doc.Text,
				Score:   res.Score,
			})
		}

		var jsonBM25Results []Result
		for _, res := range bm25Results {
			doc, ok := docLookup.Lookup(res.DocID)
			if !ok {
				continue
			}
			jsonBM25Results = append(jsonBM25Results, Result{
				ID:      fmt.Sprint(res.DocID),
				Title:   doc.Title,
				URL:     doc.URL,
				Content: doc.Text,
				Score:   res.Score,
			})
		}

		resp := Response{
			SemanticResults: jsonSemanticResults,
			BM25Results:     jsonBM25Results,
			Total:           total,
			Duration:        duration.String(),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	// separate endpoint for AI answers (runs parallel to search)
	http.HandleFunc("/api/ai-answer", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("q")
		if query == "" {
			http.Error(w, "Missing query parameter 'q'", http.StatusBadRequest)
			return
		}

		if aiClient == nil {
			http.Error(w, "AI client not available", http.StatusServiceUnavailable)
			return
		}
		aiCtx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		aiAnswer, err := aiClient.GenerateAnswer(aiCtx, query)
		if err != nil {
			log.Printf("Failed to generate AI answer: %v", err)
			http.Error(w, "AI answer generation failed", http.StatusInternalServerError)
			return
		}

		type AIResponse struct {
			Answer string `json:"answer"`
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(AIResponse{Answer: aiAnswer})
	})

	server := &http.Server{
		Addr: ":8080",
	}

	go func() {
		fmt.Println("Server started at http://localhost:8080")
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	<-ctx.Done()
	fmt.Println("\nShutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	fmt.Println("Server stopped gracefully")
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
