package quiz

import (
	"errors"
	"log/slog"
	"math/rand/v2"
	"slices"
	"strings"
)

// blank marks where the answer goes in a usage prompt.
const blank = "＿＿"

// Draft is one entry of a provider's reply: the parts of a question the
// model writes. Prompt and Answer are ignored when the target holds them.
type Draft struct {
	Code        string   `json:"code"`
	Type        Type     `json:"type"`
	Prompt      string   `json:"prompt"`
	Answer      string   `json:"answer"`
	Distractors []string `json:"distractors"`
	Explanation string   `json:"explanation"`
}

// Question is one multiple-choice question; Answer is the index of the
// correct choice.
type Question struct {
	Code        string   `json:"code"`
	Type        Type     `json:"type"`
	Prompt      string   `json:"prompt"`
	Choices     []string `json:"choices"`
	Answer      int      `json:"answer"`
	Explanation string   `json:"explanation"`
}

// Quiz is the response body; Dropped counts the targets left without a
// question.
type Quiz struct {
	Questions []Question `json:"questions"`
	Dropped   int        `json:"dropped"`
}

// assemble builds one question per target from the drafts, in target order.
// A target without exactly one draft, or whose draft breaks a rule, is
// dropped and logged; drafts for anything else are ignored.
func assemble(targets []Target, drafts []Draft) Quiz {
	quiz := Quiz{Questions: []Question{}}
	for _, t := range targets {
		var matches []Draft
		for _, d := range drafts {
			if d.Code == t.Code && d.Type == t.Type {
				matches = append(matches, d)
			}
		}
		var q Question
		err := errors.New("not exactly one entry")
		if len(matches) == 1 {
			q, err = question(t, matches[0])
		}
		if err != nil {
			slog.Warn("quiz question dropped", "code", t.Code, "type", t.Type, "reason", err)
			quiz.Dropped++
			continue
		}
		quiz.Questions = append(quiz.Questions, q)
	}
	return quiz
}

// question checks a draft against its target's rules and builds the
// question, with the choices in random order.
func question(t Target, d Draft) (Question, error) {
	prompt, answer := t.Prompt, t.Answer
	if prompt == "" {
		prompt = strings.TrimSpace(d.Prompt)
	}
	if answer == "" {
		answer = strings.TrimSpace(d.Answer)
	}
	if len(d.Distractors) != 3 {
		return Question{}, errors.New("not 3 distractors")
	}
	choices := []string{answer}
	for _, s := range d.Distractors {
		choices = append(choices, strings.TrimSpace(s))
	}
	notKana := func(s string) bool { return !isKana(s) }
	notJapanese := func(s string) bool { return !hasJapanese(s) }
	var broken string
	switch {
	case slices.Contains(choices, ""):
		broken = "an empty choice"
	case len(slices.Compact(slices.Sorted(slices.Values(choices)))) != len(choices):
		broken = "a repeated choice"
	case strings.TrimSpace(d.Explanation) == "":
		broken = "no explanation"
	case prompt == "":
		broken = "no prompt"
	case t.Examples != nil && !slices.Contains(t.Examples, prompt):
		broken = "a prompt outside the examples"
	case t.Kana && slices.ContainsFunc(choices, notKana):
		broken = "a choice that is not kana"
	case t.Blank && strings.Count(prompt, blank) != 1:
		broken = "a prompt without exactly one " + blank
	case t.Blank && slices.ContainsFunc(choices, notJapanese):
		broken = "a choice that is not Japanese"
	case t.Avoid != "" && strings.Contains(prompt, t.Avoid):
		broken = "a prompt that holds the answer"
	}
	if broken != "" {
		return Question{}, errors.New(broken)
	}
	rand.Shuffle(len(choices), func(i, j int) { choices[i], choices[j] = choices[j], choices[i] })
	return Question{Code: t.Code, Type: t.Type, Prompt: prompt, Choices: choices,
		Answer: slices.Index(choices, answer), Explanation: strings.TrimSpace(d.Explanation)}, nil
}
