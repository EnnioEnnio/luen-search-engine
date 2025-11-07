package output

import (
	"fmt"
	"strings"

	"luen-search-engine/internal/data"
	"luen-search-engine/internal/search"
)

const (
	colorReset  = "\033[0m"
	colorCyan   = "\033[36m"
	colorYellow = "\033[33m"
)

// PrintResults pretty-prints a list of results together with document metadata.
func PrintResults(results []search.Result, dataset *data.Dataset, total int) {
	if total == 0 {
		fmt.Println("No results found. Try another search term :)")
		return
	}

	fmt.Printf("\n📊 Found %d result(s)\n", total)
	fmt.Printf("%s%s%s\n", colorCyan, strings.Repeat("═", 80), colorReset)

	for i, result := range results {
		doc, ok := dataset.ByID[result.DocID]
		if !ok {
			continue
		}

		fmt.Printf("\n[%d] %s\n", i+1, doc.Title)
		fmt.Printf("    🔗 %s\n", doc.URL)

		// Print match details
		fmt.Println("    📝 Matches:")
		for _, match := range result.Matches {
			fmt.Printf("       %s count: %d\n", match.Token, match.Frequency)
		}

		if i < len(results)-1 {
			fmt.Printf("%s%s%s\n", colorYellow, strings.Repeat("─", 80), colorReset)
		}
	}

	fmt.Printf("%s%s%s\n", colorCyan, strings.Repeat("═", 80), colorReset)
}
