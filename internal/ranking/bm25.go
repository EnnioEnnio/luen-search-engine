package ranking

import "math"

// CalculateFieldedBM25Score computes BM25 with separate scoring for title and body fields.
// titleTF and bodyTF are the term frequencies in title and body respectively.
// titleLength and bodyLength are the field lengths for the specific document.
// avgTitleLength and avgBodyLength are the average field lengths across all documents.
func CalculateFieldedBM25Score(titleTF, bodyTF int, titleLength, bodyLength uint32, avgTitleLength, avgBodyLength float64, docFreq int, totalDocs int) float64 {
	b := 0.75
	k1 := 1.2
	boostTitle := 2.0
	boostBody := 1.0

	idf := CalculateIDF(totalDocs, docFreq)

	// BM25 for title field
	var titleScore float64
	if titleTF > 0 && avgTitleLength > 0 {
		titleScore = (float64(titleTF) * (k1 + 1)) /
			(float64(titleTF) + k1*(1-b+(b*float64(titleLength))/avgTitleLength))
	}

	// BM25 for body field
	var bodyScore float64
	if bodyTF > 0 && avgBodyLength > 0 {
		bodyScore = (float64(bodyTF) * (k1 + 1)) /
			(float64(bodyTF) + k1*(1-b+(b*float64(bodyLength))/avgBodyLength))
	}

	// Combine with boost factors
	return idf * (boostTitle*titleScore + boostBody*bodyScore)
}

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
