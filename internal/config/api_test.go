package config

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func validAPIEnv() map[string]string {
	vars := validEnv()
	vars["APP_PORT"] = "8080"
	vars["APP_URL"] = "http://127.0.0.1:8080"
	return vars
}

func TestLoadAPI(t *testing.T) {
	got, err := LoadAPI(env(validAPIEnv()))
	if err != nil {
		t.Fatal(err)
	}
	want := API{
		Database: Database{Host: "127.0.0.1", Port: 5432, Name: "nihongo_foundation", User: "postgres", SSLMode: "disable"},
		Port:     8080,
		URL:      "http://127.0.0.1:8080",
	}
	want.AI.Timeout = 20 * time.Second
	want.AI.OpenRouter, want.AI.OpenCodeZen = defaultChats()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v\nwant %+v", got, want)
	}

	vars := validAPIEnv()
	vars["APP_URL"] = "https://api.example"
	if got, err := LoadAPI(env(vars)); err != nil || got.URL != "https://api.example" {
		t.Errorf("APP_URL without a port: got %q, %v; want it accepted", got.URL, err)
	}
}

func TestLoadAPIRejectsMissingOrBadValues(t *testing.T) {
	for _, tt := range []struct{ key, value, want string }{
		{"DB_HOST", "", "DB_HOST is not set"},
		{"APP_PORT", "", "APP_PORT is not set; copy .env.example to .env"},
		{"APP_URL", "", "APP_URL is not set; copy .env.example to .env"},
		{"APP_PORT", "abc", `APP_PORT "abc" is not a port number`},
		{"APP_PORT", "0", `APP_PORT "0" is not a port number`},
		{"APP_PORT", "65536", `APP_PORT "65536" is not a port number`},
		{"APP_URL", "localhost", `APP_URL "localhost" is not an absolute http or https URL`},
		{"APP_URL", "ftp://api.example", `APP_URL "ftp://api.example" is not an absolute http or https URL`},
		{"APP_URL", "http://127.0.0.1:9999", `APP_URL "http://127.0.0.1:9999" names port 9999, but APP_PORT is 8080`},
		{"AI_PROVIDERS", "gpt", `AI_PROVIDERS names "gpt"; use openrouter, opencode_zen, or mock`},
	} {
		vars := validAPIEnv()
		vars[tt.key] = tt.value
		_, err := LoadAPI(env(vars))
		if err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s=%q: error = %v, want %q", tt.key, tt.value, err, tt.want)
		}
	}
}
