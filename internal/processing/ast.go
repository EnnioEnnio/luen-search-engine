package processing

import (
	"fmt"

	"luen-search-engine/internal/index"
	"luen-search-engine/internal/model"
)

type Result = model.Result
type Match = model.Match
type Token = model.Token
type DocID = model.DocID

// NodeType identifies the concrete kind of AST node.
type NodeType int

const (
	tTermNode NodeType = iota
	tPhraseNode
	tNotNode
	tAndNode
	tOrNode
)

// Node represents a query AST element that can evaluate itself against the index.
type Node interface {
	Type() NodeType
	Eval(idx index.InvertedIndex) ([]Result, error)
}

// TermNode matches a single normalized token.
type TermNode struct {
	Token Token
}

// PhraseNode matches multiple contiguous tokens in order.
type PhraseNode struct {
	Tokens []Token
}

// AndNode requires all child expressions to hold for a document.
type AndNode struct {
	Children []Node
}

// OrNode matches when any of its child expressions match.
type OrNode struct {
	Children []Node
}

// NotNode negates the result of its single child expression.
type NotNode struct {
	Child Node
}

func (n *TermNode) Type() NodeType { return tTermNode }

// Eval returns postings for the term directly from the inverted index.
func (n *TermNode) Eval(idx index.InvertedIndex) ([]Result, error) {
	if n == nil || n.Token == "" {
		return nil, nil
	}

	posting, ok := idx[n.Token]
	if !ok || posting == nil {
		return nil, nil
	}

	results := make([]Result, 0, len(posting.Docs))
	for docID, positions := range posting.Docs {
		match := Match{
			Token:     n.Token,
			Frequency: len(positions),
			Positions: positions,
		}
		results = append(results, Result{
			DocID:              docID,
			TotalTermFrequency: match.Frequency,
			Matches:            []Match{match},
		})
	}
	return results, nil
}

func (n *PhraseNode) Type() NodeType { return tPhraseNode }

// Eval performs a positional merge to find contiguous matches for the phrase.
func (n *PhraseNode) Eval(idx index.InvertedIndex) ([]Result, error) {
	if len(n.Tokens) == 0 {
		return nil, nil
	}

	docLists := make([]map[DocID]Match, len(n.Tokens))
	for ti, token := range n.Tokens {
		posting, ok := idx[token]
		if !ok || posting == nil {
			return nil, nil
		}

		docs := make(map[DocID]Match, len(posting.Docs))
		for docID, positions := range posting.Docs {
			docs[docID] = Match{
				Token:     token,
				Frequency: len(positions),
				Positions: positions,
			}
		}
		docLists[ti] = docs
	}

	results := make([]Result, 0)
	tokenCount := len(docLists)

	for docID := range docLists[0] {
		positions := make([][]int, tokenCount)
		tokens := make([]Token, tokenCount)
		missing := false

		for ti := 0; ti < tokenCount; ti++ {
			match, ok := docLists[ti][docID]
			if !ok {
				missing = true
				break
			}
			positions[ti] = match.Positions
			tokens[ti] = match.Token
		}
		if missing {
			continue
		}

		outMatches := make([]Match, tokenCount)
		for ti := 0; ti < tokenCount; ti++ {
			outMatches[ti] = Match{Token: tokens[ti], Positions: make([]int, 0)}
		}

		idxs := make([]int, tokenCount)

	OUTER:
		for {
			for ti := 0; ti < tokenCount; ti++ {
				if idxs[ti] >= len(positions[ti]) {
					break OUTER
				}
			}

			currentPositions := make([]int, tokenCount)
			for ti := 0; ti < tokenCount; ti++ {
				currentPositions[ti] = positions[ti][idxs[ti]]
			}

			ok := true
			for ti := 1; ti < tokenCount; ti++ {
				if currentPositions[ti] != currentPositions[0]+ti {
					ok = false
					break
				}
			}

			if ok {
				for ti := 0; ti < tokenCount; ti++ {
					outMatches[ti].Positions = append(outMatches[ti].Positions, currentPositions[ti])
				}
				for ti := 0; ti < tokenCount; ti++ {
					idxs[ti]++
				}
				continue
			}

			minPos := currentPositions[0]
			minIdx := 0
			for ti := 1; ti < tokenCount; ti++ {
				if currentPositions[ti] < minPos {
					minPos = currentPositions[ti]
					minIdx = ti
				}
			}
			idxs[minIdx]++
		}

		if len(outMatches[0].Positions) == 0 {
			continue
		}

		total := len(outMatches[0].Positions)
		for ti := 0; ti < tokenCount; ti++ {
			outMatches[ti].Frequency = len(outMatches[ti].Positions)
		}

		results = append(results, Result{
			DocID:              docID,
			TotalTermFrequency: total,
			Matches:            outMatches,
		})
	}

	return results, nil
}

func (a *AndNode) Type() NodeType { return tAndNode }

// Eval intersects child results and subtracts explicit negations.
func (a *AndNode) Eval(idx index.InvertedIndex) ([]Result, error) {
	if len(a.Children) == 0 {
		return nil, nil
	}

	exclude := make(map[DocID]struct{})
	var current map[DocID]Result
	positiveCount := 0

	for _, child := range a.Children {
		switch node := child.(type) {
		case *NotNode:
			neg, err := node.evalNegated(idx)
			if err != nil {
				return nil, err
			}
			for _, r := range neg {
				exclude[r.DocID] = struct{}{}
			}
		default:
			res, err := child.Eval(idx)
			if err != nil {
				return nil, err
			}
			positiveCount++
			resMap := resultsToMap(res)
			if len(resMap) == 0 {
				return nil, nil
			}
			if current == nil {
				current = resMap
			} else {
				current = intersectResultMaps(current, resMap)
				if len(current) == 0 {
					return nil, nil
				}
			}
		}
	}

	if positiveCount == 0 {
		return nil, fmt.Errorf("query must contain at least one positive term")
	}

	for docID := range exclude {
		delete(current, docID)
	}

	if len(current) == 0 {
		return nil, nil
	}

	return mapToSlice(current), nil
}

func (o *OrNode) Type() NodeType { return tOrNode }

// Eval unions child results, summing frequencies per document.
func (o *OrNode) Eval(idx index.InvertedIndex) ([]Result, error) {
	if len(o.Children) == 0 {
		return nil, nil
	}

	aggregate := make(map[DocID]Result)

	for _, child := range o.Children {
		res, err := child.Eval(idx)
		if err != nil {
			return nil, err
		}

		for _, r := range res {
			existing, ok := aggregate[r.DocID]
			if !ok {
				aggregate[r.DocID] = Result{
					DocID:              r.DocID,
					TotalTermFrequency: r.TotalTermFrequency,
					Matches:            append([]Match(nil), r.Matches...),
				}
				continue
			}
			existing.TotalTermFrequency += r.TotalTermFrequency
			existing.Matches = append(existing.Matches, r.Matches...)
			aggregate[r.DocID] = existing
		}
	}

	if len(aggregate) == 0 {
		return nil, nil
	}

	return mapToSlice(aggregate), nil
}

func (n *NotNode) Type() NodeType { return tNotNode }

// Eval always errors—NOT must be combined with a positive operand via AndNode.
func (n *NotNode) Eval(idx index.InvertedIndex) ([]Result, error) {
	return nil, fmt.Errorf("NOT expressions must be combined with a positive search term")
}

// evalNegated returns the child results for exclusion handling.
func (n *NotNode) evalNegated(idx index.InvertedIndex) ([]Result, error) {
	if n == nil || n.Child == nil {
		return nil, fmt.Errorf("negation is missing an operand")
	}
	return n.Child.Eval(idx)
}

// resultsToMap converts a slice of results into a docID keyed map.
func resultsToMap(results []Result) map[DocID]Result {
	if len(results) == 0 {
		return nil
	}

	out := make(map[DocID]Result, len(results))
	for _, r := range results {
		out[r.DocID] = r
	}
	return out
}

// intersectResultMaps keeps documents present in both maps and merges matches.
func intersectResultMaps(a, b map[DocID]Result) map[DocID]Result {
	if len(a) == 0 || len(b) == 0 {
		return map[DocID]Result{}
	}

	if len(a) > len(b) {
		a, b = b, a
	}

	out := make(map[DocID]Result, len(a))
	for docID, left := range a {
		right, ok := b[docID]
		if !ok {
			continue
		}
		merged := Result{
			DocID:              docID,
			TotalTermFrequency: left.TotalTermFrequency + right.TotalTermFrequency,
			Matches:            append(append([]Match(nil), left.Matches...), right.Matches...),
		}
		out[docID] = merged
	}
	return out
}

// mapToSlice flattens a map of results into an arbitrarily ordered slice.
func mapToSlice(m map[DocID]Result) []Result {
	if len(m) == 0 {
		return nil
	}
	out := make([]Result, 0, len(m))
	for _, r := range m {
		out = append(out, r)
	}
	return out
}
