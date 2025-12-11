package ranking

import "math"

// calculateBM25Term computes the BM25 term component (without IDF).
func calculateBM25Term(termFreq int, docLength uint32, avgDocLength float64, k1, b float64) float64 {
	if termFreq == 0 || avgDocLength == 0 {
		return 0
	}
	return (float64(termFreq) * (k1 + 1)) /
		(float64(termFreq) + k1*(1-b+(b*float64(docLength))/(avgDocLength)))
}

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

	// Field-normalized term frequencies
	var normalizedTitleTF float64
	if avgTitleLength > 0 {
		normalizedTitleTF = float64(titleTF) / (1 - b + b*(float64(titleLength)/avgTitleLength))
	}

	var normalizedBodyTF float64
	if avgBodyLength > 0 {
		normalizedBodyTF = float64(bodyTF) / (1 - b + b*(float64(bodyLength)/avgBodyLength))
	}

	// Combined field-normalized TF with boost factors
	combinedTF := boostTitle*normalizedTitleTF + boostBody*normalizedBodyTF

	// Apply BM25 transformation to combined TF
	if combinedTF == 0 {
		return 0
	}
	return idf * (combinedTF * (k1 + 1)) / (combinedTF + k1)
}

func CalculateBM25Score(termFreq int, docLength uint32, avgDocLength float64, docFreq int, totalDocs int) float64 {
	b := 0.75
	k1 := 1.2

	idf := CalculateIDF(totalDocs, docFreq)
	bm25Term := calculateBM25Term(termFreq, docLength, avgDocLength, k1, b)

	return idf * bm25Term
}

// CalculateIDF computes the inverse document frequency for a term
func CalculateIDF(totalDocs int, docFreq int) float64 {
	return math.Log((float64(totalDocs) - float64(docFreq) + 0.5) /
		(float64(docFreq) + 0.5))
}
