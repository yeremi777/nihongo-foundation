package config

import (
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"
)

// maxAITimeout is kept under the API's 25s shutdown drain.
const maxAITimeout = 20 * time.Second

// providerNames are the names AI_PROVIDERS may list.
var providerNames = []string{"openrouter", "opencode_zen", "mock"}

// AI is how quizzes ask AI providers: the provider names in fall-through
// order, the deadline of one quiz's provider chain, and each chat provider's
// settings.
type AI struct {
	Providers   []string
	Timeout     time.Duration
	OpenRouter  Chat
	OpenCodeZen Chat
}

// Chat is one OpenAI-compatible chat completions provider; Headers are sent
// with every request.
type Chat struct {
	Name      string
	ServerURL string
	APIKey    string
	Model     string
	Headers   map[string]string
}

// LoadAI reads the AI_*, OPENROUTER_*, and OPENCODE_ZEN_* variables, every one
// optional. A server URL is checked only for a listed provider.
func LoadAI(getenv func(string) string) (AI, error) {
	var providers []string
	for _, name := range strings.Split(getenv("AI_PROVIDERS"), ",") {
		name = strings.ToLower(strings.TrimSpace(name))
		switch {
		case name == "":
			continue
		case !slices.Contains(providerNames, name):
			return AI{}, fmt.Errorf("AI_PROVIDERS names %q; use openrouter, opencode_zen, or mock", name)
		case slices.Contains(providers, name):
			return AI{}, fmt.Errorf("AI_PROVIDERS names %q twice", name)
		}
		providers = append(providers, name)
	}
	if len(providers) > 1 && slices.Contains(providers, "mock") {
		return AI{}, fmt.Errorf("AI_PROVIDERS lists mock with another provider; mock runs alone")
	}

	timeout := maxAITimeout
	if raw := getenv("AI_TIMEOUT_SECONDS"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || time.Duration(n)*time.Second > maxAITimeout {
			return AI{}, fmt.Errorf("AI_TIMEOUT_SECONDS %q is not a number of seconds from 1 to %d", raw, int(maxAITimeout.Seconds()))
		}
		timeout = time.Duration(n) * time.Second
	}

	ai := AI{
		Providers: providers,
		Timeout:   timeout,
		OpenRouter: Chat{
			Name:      "OpenRouter",
			ServerURL: or(getenv("OPENROUTER_SERVER_URL"), "https://openrouter.ai/api/v1"),
			APIKey:    apiKey(getenv("OPENROUTER_API_KEY")),
			Model:     or(getenv("OPENROUTER_MODEL"), "openrouter/free"),
			Headers: map[string]string{
				"HTTP-Referer":       or(getenv("OPENROUTER_HTTP_REFERER"), "http://127.0.0.1:8080"),
				"X-OpenRouter-Title": or(getenv("OPENROUTER_APP_TITLE"), "Nihongo Foundation"),
			},
		},
		OpenCodeZen: Chat{
			Name:      "OpenCode Zen",
			ServerURL: or(getenv("OPENCODE_ZEN_SERVER_URL"), "https://opencode.ai/zen/v1"),
			APIKey:    apiKey(getenv("OPENCODE_ZEN_API_KEY")),
			Model:     strings.TrimPrefix(getenv("OPENCODE_ZEN_MODEL"), "opencode/"),
		},
	}
	for _, p := range []struct{ name, key, value string }{
		{"openrouter", "OPENROUTER_SERVER_URL", ai.OpenRouter.ServerURL},
		{"opencode_zen", "OPENCODE_ZEN_SERVER_URL", ai.OpenCodeZen.ServerURL},
	} {
		if !slices.Contains(providers, p.name) {
			continue
		}
		if u, err := url.Parse(p.value); err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return AI{}, fmt.Errorf("%s %q is not an absolute http or https URL", p.key, p.value)
		}
	}
	return ai, nil
}

func or(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

// apiKey treats a placeholder such as <your-key> as no key.
func apiKey(key string) string {
	if strings.HasPrefix(key, "<") {
		return ""
	}
	return key
}
