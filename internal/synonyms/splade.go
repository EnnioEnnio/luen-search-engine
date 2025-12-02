package synonyms

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/nlpodyssey/cybertron/pkg/tasks"
	"github.com/nlpodyssey/cybertron/pkg/tasks/languagemodeling"
	lmbert "github.com/nlpodyssey/cybertron/pkg/tasks/languagemodeling/bert"
	"github.com/nlpodyssey/spago/mat"

	"luen-search-engine/internal/processing"
)

type SpladeLike struct {
	lm *lmbert.LanguageModel
}

func NewSpladeLike() (*SpladeLike, error) {
	conf := &tasks.Config{
		ModelsDir:        "models",
		ModelName:        "bert-base-cased",
		DownloadPolicy:   tasks.DownloadMissing,
		ConversionPolicy: tasks.ConvertMissing,
	}

	// Sorgt dafür, dass das Modell überhaupt existiert (Download/Conversion)
	_, err := tasks.Load[languagemodeling.Interface](conf)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare model: %w", err)
	}

	modelDir := conf.FullModelPath() // ergibt "models/bert-base-cased"

	lm, err := lmbert.LoadMaskedLanguageModel(modelDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load masked LM: %w", err)
	}

	return &SpladeLike{lm: lm}, nil
}

type Synonym struct {
	Word   string
	Weight float64
}

func (s *SpladeLike) findTokenPositions(query, term string) []int {
	tokens := s.lm.Tokenizer.Tokenize(query)
	positions := []int{}
	lowerTerm := strings.ToLower(term)

	for i, tok := range tokens {
		if strings.ToLower(tok) == lowerTerm {
			positions = append(positions, i)
		}
	}
	return positions
}

func (s *SpladeLike) logitsForMaskedPosition(ctx context.Context, tokens []string, idx int) ([]float32, error) {
	// Kopie der Tokens mit einer [MASK] an Stelle idx
	masked := make([]string, len(tokens))
	copy(masked, tokens)
	masked[idx] = "[MASK]"

	preds, err := s.lm.Model.Predict(ctx, masked)
	if err != nil {
		return nil, fmt.Errorf("MLM Predict failed: %w", err)
	}

	tensor, ok := preds[idx]
	if !ok {
		return nil, fmt.Errorf("no logits for masked index %d", idx)
	}

	logits := mat.Data[float32](tensor) // Länge = vocab_size
	return logits, nil
}

func spladeTransform(logits []float32) []float64 {
	weights := make([]float64, len(logits))
	for j, v := range logits {
		x := float64(v)
		if x < 0 {
			x = 0 // ReLU
		}
		weights[j] = math.Log(1 + x)
	}
	return weights
}

func (s *SpladeLike) buildVocabIndex() []string {
	vocabMap := s.lm.Tokenizer.Vocabulary() // z.B. map[string]int
	vocab := make([]string, len(vocabMap))
	for token, id := range vocabMap {
		if id >= 0 && id < len(vocab) {
			vocab[id] = token
		}
	}
	return vocab
}

func (s *SpladeLike) topKSynonymsFromWeights(weights []float64, fullQuery string, term string, k int) []Synonym {
	vocab := s.buildVocabIndex()

	type pair struct {
		word   string
		weight float64
	}

	pairs := make([]pair, 0, len(weights))
	lowerQuery := strings.ToLower(fullQuery)
	lowerTerm := strings.ToLower(term)

	for id, w := range weights {
		if w <= 0 {
			continue
		}
		if id < 0 || id >= len(vocab) {
			continue
		}
		word := vocab[id]
		if word == "" {
			continue
		}

		lw := strings.ToLower(word)

		// Originalwort und Tokens, die schon in der Query vorkommen, rauswerfen
		if lw == lowerTerm {
			continue
		}
		if strings.Contains(lowerQuery, lw) {
			continue
		}
		// Sondertokens etc. filtern
		if lw == "[cls]" || lw == "[sep]" || lw == "[pad]" {
			continue
		}
		if strings.HasPrefix(word, "##") {
			continue
		}

		pairs = append(pairs, pair{word: word, weight: w})
	}

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].weight > pairs[j].weight
	})

	if len(pairs) > k {
		pairs = pairs[:k]
	}

	syns := make([]Synonym, len(pairs))
	for i, p := range pairs {
		syns[i] = Synonym{Word: p.word, Weight: p.weight}
	}
	return syns
}

func (s *SpladeLike) PredictSynonyms(ctx context.Context, fullQuery, term string, topK int) ([]Synonym, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	tokens := s.lm.Tokenizer.Tokenize(fullQuery)
	positions := s.findTokenPositions(fullQuery, term)
	if len(positions) == 0 {
		// Fallback: gar nichts expandieren
		return nil, nil
	}

	// Für mehrere Positionen des Terms könntest du hier über i max-poolen.
	// Wir nehmen erstmal nur die erste Vorkommens-Position.
	idx := positions[0]

	logits, err := s.logitsForMaskedPosition(ctx, tokens, idx)
	if err != nil {
		return nil, err
	}

	weights := spladeTransform(logits)
	syns := s.topKSynonymsFromWeights(weights, fullQuery, term, topK)

	return syns, nil
}

func (s *SpladeLike) ExpandAST(ctx context.Context, ast processing.Node, fullQuery string) (processing.Node, error) {
	expandTermNode := func(termNode *processing.TermNode) (*processing.OrNode, error) {
		syns, err := s.PredictSynonyms(ctx, fullQuery, string(termNode.Token), 5)
		if err != nil {
			return nil, err
		}

		children := make([]processing.Node, 0, len(syns)+1)
		children = append(children, termNode) // Originalterm

		for _, syn := range syns {
			children = append(children, &processing.TermNode{
				Token: processing.Token(syn.Word),
			})
		}
		return &processing.OrNode{Children: children}, nil
	}

	switch node := ast.(type) {
	case *processing.TermNode:
		return expandTermNode(node)

	case *processing.PhraseNode:
		return node, nil

	case *processing.AndNode:
		expandedChildren := make([]processing.Node, len(node.Children))
		for i, child := range node.Children {
			expandedChild, err := s.ExpandAST(ctx, child, fullQuery)
			if err != nil {
				return nil, err
			}
			expandedChildren[i] = expandedChild
		}
		return &processing.AndNode{Children: expandedChildren}, nil

	case *processing.OrNode:
		expandedChildren := make([]processing.Node, len(node.Children))
		for i, child := range node.Children {
			expandedChild, err := s.ExpandAST(ctx, child, fullQuery)
			if err != nil {
				return nil, err
			}
			expandedChildren[i] = expandedChild
		}
		return &processing.OrNode{Children: expandedChildren}, nil

	case *processing.NotNode:
		if termNode, ok := node.Child.(*processing.TermNode); ok {
			expandedChildren, err := expandTermNode(termNode)
			if err != nil {
				return nil, err
			}
			for i, child := range expandedChildren.Children {
				expandedChildren.Children[i] = &processing.NotNode{Child: child}
			}
			return expandedChildren, nil
		}
		expandedChild, err := s.ExpandAST(ctx, node.Child, fullQuery)
		if err != nil {
			return nil, err
		}
		return &processing.NotNode{Child: expandedChild}, nil

	default:
		return nil, fmt.Errorf("unsupported node type: %T", node)
	}
}
