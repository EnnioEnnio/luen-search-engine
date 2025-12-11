package ranking

import "math"

func CalculateBM25Score(termFreq int, docLength uint32, avgDocLength float64, docFreq int, totalDocs int) float64 {
	b := 0.75
	k1 := 1.2

	idf := CalculateIDF(totalDocs, docFreq)

	inner := (float64(termFreq) * (k1 + 1)) /
		(float64(termFreq) + k1*(1-b+(b*float64(docLength))/(avgDocLength)))

	bm25 := idf * inner

	return bm25
}

// CalculateIDF computes the inverse document frequency for a term
func CalculateIDF(totalDocs int, docFreq int) float64 {
	return math.Log((float64(totalDocs) - float64(docFreq) + 0.5) /
		(float64(docFreq) + 0.5))
}
