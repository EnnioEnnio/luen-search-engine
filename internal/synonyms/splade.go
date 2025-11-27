package synonyms

import (
	"context"
	"fmt"
	"strings"

	"luen-search-engine/internal/processing"

	"github.com/nlpodyssey/cybertron/pkg/tasks"
	"github.com/nlpodyssey/cybertron/pkg/tasks/languagemodeling"
	"github.com/rs/zerolog"
)

type SpladeLike struct {
	model languagemodeling.Interface
}

func NewSpladeLike() (*SpladeLike, error) {
	zerolog.SetGlobalLevel(zerolog.PanicLevel) // suppress logs from cybertron

	conf := &tasks.Config{
		ModelsDir: "models",
		ModelName: "bert-base-cased", // in the paper they use "distilbert-base-uncased". But this is not available in cybertron. The original Splade implementation uses "bert-base-uncased".But we use "bert-base-cased" as we need this for the MASK token to work properly.
	}

	m, err := tasks.Load[languagemodeling.Interface](conf)
	if err != nil {
		return nil, fmt.Errorf("failed to load model %s: %w", conf.ModelName, err)
	}

	return &SpladeLike{model: m}, nil
}

func (s *SpladeLike) PredictSynonyms(ctx context.Context, fullQuery string, term string) ([]string, error) {
	maskedPrompt := fmt.Sprintf(
		`In "%s", "%s" and [MASK] are related`,
		fullQuery, term,
	)

	resp, err := s.model.Predict(ctx, maskedPrompt, languagemodeling.Parameters{K: 2})
	if err != nil {
		return nil, fmt.Errorf("predict failed for %q: %w", term, err)
	}
	if len(resp.Tokens) == 0 {
		return []string{}, nil
	}
	synonyms := make([]string, 0)
	for _, w := range resp.Tokens[0].Words {
		if strings.Contains(fullQuery, w) {
			continue
		}
		synonyms = append(synonyms, w)
	}
	return synonyms, nil
}

func (s *SpladeLike) ExpandAST(ctx context.Context, ast processing.Node, fullQuery string) (processing.Node, error) {
	expandTermNode := func(termNode *processing.TermNode) (*processing.OrNode, error) {
		synonyms, err := s.PredictSynonyms(ctx, fullQuery, string(termNode.Token))
		if err != nil {
			return nil, err
		}

		children := make([]processing.Node, 0, len(synonyms)+1)
		children = append(children, termNode)
		for _, synonym := range synonyms {
			children = append(children, &processing.TermNode{Token: processing.Token(synonym)})
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
