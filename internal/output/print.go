package output

import (
	"fmt"
	"strings"
	"time"

	"luen-search-engine/internal/data"
	"luen-search-engine/internal/model"
)

type Result = model.Result

const (
	colorReset  = "\033[0m"
	colorCyan   = "\033[36m"
	colorYellow = "\033[33m"
)

// PrintResults pretty-prints a list of results together with document metadata.
func PrintResults(semanticResults []Result, bm25Results []Result, lookup *data.DocumentStore, total int, searchTime time.Duration, DocLengths map[model.DocID]model.FieldDocLengths) {
	if len(bm25Results) == 0 && len(semanticResults) == 0 {
		fmt.Println("No results found. Try another search term :)")
		return
	}

	fmt.Printf("\n📊 Found %d result(s) in %s\n", total, searchTime)
	fmt.Println("\n🔍 BM25 Search Results:")
	fmt.Printf("%s%s%s\n", colorCyan, strings.Repeat("═", 80), colorReset)
	if len(bm25Results) == 0 {
		fmt.Println("No BM25 results found.")
	}

	for i, result := range bm25Results {
		doc, ok := lookup.Lookup(result.DocID)
		if !ok {
			continue
		}
		lastResult := false
		if i == len(bm25Results)-1 {
			lastResult = true
		}
		printReult(result, i, doc, DocLengths, lastResult)
	}

	fmt.Printf("%s%s%s\n", colorCyan, strings.Repeat("═", 80), colorReset)
	fmt.Println("\n🤖 Semantic Search Results:")
	if len(semanticResults) == 0 {
		fmt.Println("No semantic results found.")
	} else {
		for i, sResult := range semanticResults {
			doc, ok := lookup.Lookup(sResult.DocID)
			if !ok {
				continue
			}
			lastResult := false
			if i == len(semanticResults)-1 {
				lastResult = true
			}
			printReult(sResult, i, doc, DocLengths, lastResult)
		}
	}

	fmt.Printf("%s%s%s\n", colorCyan, strings.Repeat("═", 80), colorReset)
}

func printReult(result Result, i int, doc data.Document, DocLengths map[model.DocID]model.FieldDocLengths, lastResult bool) {
	fmt.Printf("\n[%d] %s\n", i+1, doc.Title)
	fmt.Printf("    🔗 %s\n", doc.URL)

	// Print match details
	if len(result.Matches) > 0 {
		fmt.Println("    📝 Matches:")
	}
	for _, match := range result.Matches {
		fmt.Printf("       %s count: %d\n", match.Token, match.Frequency)
	}
	fmt.Printf("    📏 Document length: %d\n", DocLengths[result.DocID].BodyLength+DocLengths[result.DocID].TitleLength)
	fmt.Printf("    ⭐ Score: %.4f\n", result.Score)

	if !lastResult {
		fmt.Printf("%s%s%s\n", colorYellow, strings.Repeat("─", 80), colorReset)
	}
}
