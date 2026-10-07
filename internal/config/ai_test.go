package config

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

// defaultChats are the provider settings with no OPENROUTER_* or
// OPENCODE_ZEN_* variable set.
func defaultChats() (Chat, Chat) {
	return Chat{Name: "OpenRouter", ServerURL: "https://openrouter.ai/api/v1", Model: "openrouter/free",
			Headers: map[string]string{"HTTP-Referer": "http://127.0.0.1:8080", "X-OpenRouter-Title": "Nihongo Foundation"}},
		Chat{Name: "OpenCode Zen", ServerURL: "https://opencode.ai/zen/v1"}
}

func TestLoadAI(t *testing.T) {
	openRouter, zen := defaultChats()
	got, err := LoadAI(env(map[string]string{}))
	if want := (AI{Timeout: 20 * time.Second, OpenRouter: openRouter, OpenCodeZen: zen}); err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("nothing set: got %+v, %v\nwant %+v", got, err, want)
	}

	got, err = LoadAI(env(map[string]string{
		"AI_PROVIDERS": " OpenRouter , opencode_zen,", "AI_TIMEOUT_SECONDS": "15",
		"OPENROUTER_API_KEY": "sk-or", "OPENROUTER_MODEL": "openai/gpt-4.1-mini", "OPENROUTER_SERVER_URL": "http://127.0.0.1:9000/v1",
		"OPENROUTER_APP_TITLE": "Nihongo", "OPENROUTER_HTTP_REFERER": "https://nihongo.example",
		"OPENCODE_ZEN_API_KEY": "<your-key>", "OPENCODE_ZEN_MODEL": "opencode/grok-code",
	}))
	want := AI{
		Providers: []string{"openrouter", "opencode_zen"},
		Timeout:   15 * time.Second,
		OpenRouter: Chat{Name: "OpenRouter", ServerURL: "http://127.0.0.1:9000/v1", APIKey: "sk-or", Model: "openai/gpt-4.1-mini",
			Headers: map[string]string{"HTTP-Referer": "https://nihongo.example", "X-OpenRouter-Title": "Nihongo"}},
		OpenCodeZen: Chat{Name: "OpenCode Zen", ServerURL: "https://opencode.ai/zen/v1", Model: "grok-code"},
	}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("all set:\ngot  %+v, %v\nwant %+v", got, err, want)
	}

	if got, err := LoadAI(env(map[string]string{"AI_PROVIDERS": "mock"})); err != nil || !reflect.DeepEqual(got.Providers, []string{"mock"}) {
		t.Errorf("mock: got %v, %v", got.Providers, err)
	}
}

func TestLoadAIRejectsBadValues(t *testing.T) {
	for _, tt := range []struct {
		vars map[string]string
		want string
	}{
		{map[string]string{"AI_PROVIDERS": "openai"}, `AI_PROVIDERS names "openai"; use openrouter, opencode_zen, or mock`},
		{map[string]string{"AI_PROVIDERS": "openrouter,OpenRouter"}, `AI_PROVIDERS names "openrouter" twice`},
		{map[string]string{"AI_PROVIDERS": "openrouter,mock"}, "AI_PROVIDERS lists mock with another provider; mock runs alone"},
		{map[string]string{"AI_PROVIDERS": "openrouter", "OPENROUTER_SERVER_URL": "localhost"}, `OPENROUTER_SERVER_URL "localhost" is not an absolute http or https URL`},
		{map[string]string{"AI_PROVIDERS": "opencode_zen", "OPENCODE_ZEN_SERVER_URL": "ftp://zen"}, `OPENCODE_ZEN_SERVER_URL "ftp://zen" is not an absolute http or https URL`},
		{map[string]string{"AI_TIMEOUT_SECONDS": "abc"}, `AI_TIMEOUT_SECONDS "abc" is not a number of seconds from 1 to 20`},
		{map[string]string{"AI_TIMEOUT_SECONDS": "0"}, `AI_TIMEOUT_SECONDS "0" is not a number of seconds from 1 to 20`},
		{map[string]string{"AI_TIMEOUT_SECONDS": "21"}, `AI_TIMEOUT_SECONDS "21" is not a number of seconds from 1 to 20`},
	} {
		_, err := LoadAI(env(tt.vars))
		if err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%v: error = %v, want %q", tt.vars, err, tt.want)
		}
	}

	if _, err := LoadAI(env(map[string]string{"OPENROUTER_SERVER_URL": "localhost"})); err != nil {
		t.Errorf("unlisted provider with a bad URL: error = %v, want none", err)
	}
}
