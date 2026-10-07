package quiz

import "context"

// Mock writes fixed drafts that pass every rule, with no network.
type Mock struct{}

func (Mock) Generate(_ context.Context, _ Lang, targets []Target) ([]Draft, error) {
	drafts := make([]Draft, len(targets))
	for i, t := range targets {
		d := Draft{Code: t.Code, Type: t.Type, Distractors: []string{"mock 1", "mock 2", "mock 3"}, Explanation: "Mock explanation."}
		if t.Kana || t.Blank {
			d.Distractors = []string{"もっくいち", "もっくに", "もっくさん"}
		}
		if t.Examples != nil {
			d.Prompt = t.Examples[0]
		}
		if t.Blank {
			d.Prompt = blank + "。"
		}
		if t.Answer == "" {
			d.Answer = "もっく"
		}
		drafts[i] = d
	}
	return drafts, nil
}
