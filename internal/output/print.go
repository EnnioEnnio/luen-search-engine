package output

import (
	"fmt"

	"luen-search-engine/internal/data"
	"luen-search-engine/internal/search"
)

// PrintResults pretty-prints a list of results together with document metadata.
func PrintResults(results []search.Result, dataset *data.Dataset, total int) {
	if total == 0 {
		fmt.Println("No results found. Try another search term :)")
		return
	}

	fmt.Printf("Result Count: '%d'\n\n", total)
	fmt.Println("Results:")

	for _, result := range results {
		doc, ok := dataset.ByID[result.DocID]
		if !ok {
			continue
		}

		fmt.Println(doc.URL)
		fmt.Println(doc.Title)
		for _, match := range result.Matches {
			fmt.Printf("%s count: %d\n", match.Token, match.Frequency)
		}
		fmt.Println("---------------------------")
	}
}
