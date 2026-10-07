package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/yeremi777/nihongo-foundation/internal/config"
	"github.com/yeremi777/nihongo-foundation/internal/quiz"
)

var meaningTarget = []quiz.Target{{Code: "v101-001", Type: quiz.TypeMeaning, Prompt: "あまい", Answer: "sweet"}}

// provider is a chat completions server that counts the requests it got.
func provider(t *testing.T, asked *int) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		*asked++
		_, _ = io.WriteString(w, `{"choices":[{"message":{"content":"{\"questions\":[{\"code\":\"v101-001\",\"type\":\"meaning\",`+
			`\"prompt\":\"\",\"answer\":\"\",\"distractors\":[\"salty\",\"sour\",\"bitter\"],\"explanation\":\"Amai is sweet.\"}]}"}}]}`)
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func TestGeneratorForSkipsAListedProviderWithoutKeyOrModel(t *testing.T) {
	var routerAsked, zenAsked int
	ai := config.AI{
		Providers:   []string{"openrouter", "opencode_zen"},
		OpenRouter:  config.Chat{Name: "OpenRouter", ServerURL: provider(t, &routerAsked), Model: "openrouter/free"},
		OpenCodeZen: config.Chat{Name: "OpenCode Zen", ServerURL: provider(t, &zenAsked), APIKey: "sk-zen", Model: "grok-code"},
	}
	drafts, err := generatorFor(ai).Generate(context.Background(), quiz.LangEN, meaningTarget)
	if err != nil || len(drafts) != 1 || drafts[0].Explanation != "Amai is sweet." || routerAsked != 0 || zenAsked != 1 {
		t.Errorf("got %+v, %v; OpenRouter asked %d, OpenCode Zen %d; want only OpenCode Zen asked", drafts, err, routerAsked, zenAsked)
	}

	ai.Providers = []string{"openrouter"}
	if g := generatorFor(ai); g != nil {
		t.Errorf("only a keyless provider: got %T, want no generator", g)
	}
	if g := generatorFor(config.AI{}); g != nil {
		t.Errorf("no provider: got %T, want no generator", g)
	}

	drafts, err = generatorFor(config.AI{Providers: []string{"mock"}}).Generate(context.Background(), quiz.LangEN, meaningTarget)
	if err != nil || len(drafts) != 1 || drafts[0].Explanation != "Mock explanation." {
		t.Errorf("mock: got %+v, %v", drafts, err)
	}
}
