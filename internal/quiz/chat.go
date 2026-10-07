package quiz

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
)

// maxReplyBytes bounds a provider reply read into memory.
const maxReplyBytes = 1 << 20

// fallThroughStatuses send a chain to its next provider, as any 5xx does.
var fallThroughStatuses = []int{402, 408, 409, 425, 429}

// systemPrompt holds the reply shape and the rules each entry is written to;
// the mechanical ones are checked by question.
const systemPrompt = `You write multiple-choice JLPT practice questions for the given targets.
Answer one JSON object: {"questions":[{"code":"","type":"","prompt":"","answer":"","distractors":["","",""],"explanation":""}]}, with one entry per target and its code and type copied exactly.
Every entry has 3 distractors: wrong for the item, plausible at its level, distinct from each other and from the answer.
Every entry has an explanation of why the answer is right, in English when lang is "en" and Indonesian when lang is "id"; one or two sentences.
Leave prompt and answer "" unless a rule below asks for them.
- meaning: distractors are meanings in lang, in the style of the item's meaning.
- reading of a kanji: prompt is one word copied exactly from the item's examples, preferring one without 〜; answer is that word's reading in hiragana; distractors are hiragana readings that sound close.
- reading of a word: distractors are hiragana readings that sound close to the item's reading.
- usage of a word: prompt is a natural Japanese sentence at the item's level with ＿＿ once where the word belongs, and the word nowhere else; distractors are Japanese words of the same part of speech that do not fit.
- usage of a grammar pattern: prompt is a natural Japanese sentence at the item's level with ＿＿ once where the pattern belongs; answer is the exact text that fills ＿＿; distractors are Japanese forms that do not fit.`

// fallThrough marks a provider failure after which a chain asks its next
// provider.
type fallThrough struct{ error }

func (f fallThrough) Unwrap() error { return f.error }

func fallsThrough(err error) bool {
	var f fallThrough
	return errors.As(err, &f)
}

// Chain asks its providers in order and answers the first success; a failure
// not marked to fall through, or an ended context, stops it.
type Chain []Generator

// Generate returns the drafts of the first provider that succeeds, or the
// failures of every provider asked, joined.
func (c Chain) Generate(ctx context.Context, lang Lang, targets []Target) ([]Draft, error) {
	var failures []error
	for _, g := range c {
		drafts, err := g.Generate(ctx, lang, targets)
		if err == nil {
			return drafts, nil
		}
		failures = append(failures, err)
		if !fallsThrough(err) || ctx.Err() != nil {
			break
		}
	}
	return nil, errors.Join(failures...)
}

// Chat writes drafts through an OpenAI-compatible chat completions provider.
// A call is bounded by its context.
type Chat struct {
	name    string
	url     string
	model   string
	apiKey  string
	headers map[string]string
	client  *http.Client
}

// NewChat calls <serverURL>/chat/completions with model and apiKey, sending
// headers with every request; name labels its errors.
func NewChat(name, serverURL, model, apiKey string, headers map[string]string) Chat {
	return Chat{name: name, url: strings.TrimRight(serverURL, "/") + "/chat/completions", model: model, apiKey: apiKey,
		headers: headers, client: &http.Client{}}
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type userTarget struct {
	Code string `json:"code"`
	Type Type   `json:"type"`
	Item Facts  `json:"item"`
}

// Generate asks the model for one draft per target.
func (c Chat) Generate(ctx context.Context, lang Lang, targets []Target) ([]Draft, error) {
	user := struct {
		Lang    Lang         `json:"lang"`
		Targets []userTarget `json:"targets"`
	}{Lang: lang, Targets: make([]userTarget, len(targets))}
	for i, t := range targets {
		user.Targets[i] = userTarget{Code: t.Code, Type: t.Type, Item: t.Facts}
	}
	userJSON, err := json.Marshal(user)
	if err != nil {
		return nil, err
	}
	var chat struct {
		Model          string        `json:"model"`
		Messages       []chatMessage `json:"messages"`
		ResponseFormat struct {
			Type string `json:"type"`
		} `json:"response_format"`
	}
	chat.Model = c.model
	chat.Messages = []chatMessage{{"system", systemPrompt}, {"user", string(userJSON)}}
	chat.ResponseFormat.Type = "json_object"
	body, err := json.Marshal(chat)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}
	res, err := c.client.Do(req)
	if err != nil {
		return nil, fallThrough{fmt.Errorf("%s request failed: %w", c.name, err)}
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(res.Body, maxReplyBytes))
	if err != nil {
		return nil, fallThrough{fmt.Errorf("%s reply read failed: %w", c.name, err)}
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		err := fmt.Errorf("%s answered %d: %.512s", c.name, res.StatusCode, raw)
		if res.StatusCode >= 500 || slices.Contains(fallThroughStatuses, res.StatusCode) {
			return nil, fallThrough{err}
		}
		return nil, err
	}
	drafts, err := parseReply(raw)
	if err != nil {
		return nil, fmt.Errorf("%s %w", c.name, err)
	}
	return drafts, nil
}

// parseReply reads the drafts from the first choice of a chat completions
// reply, with a markdown code fence around its content removed.
func parseReply(raw []byte) ([]Draft, error) {
	var reply struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &reply); err != nil {
		return nil, fmt.Errorf("reply: %w", err)
	}
	if len(reply.Choices) == 0 || reply.Choices[0].Message.Content == "" {
		return nil, errors.New("reply has no content")
	}
	content := unfence(reply.Choices[0].Message.Content)
	var parsed struct {
		Questions *[]Draft `json:"questions"`
	}
	if err := json.Unmarshal([]byte(content), &parsed); err != nil || parsed.Questions == nil {
		return nil, fmt.Errorf("content is not the expected JSON: %.512s", content)
	}
	return *parsed.Questions, nil
}

// unfence removes a markdown code fence around the model's JSON.
func unfence(content string) string {
	s := strings.TrimSpace(content)
	if !strings.HasPrefix(s, "```") {
		return s
	}
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(strings.TrimSpace(s), "```")
	return strings.TrimSpace(s)
}
