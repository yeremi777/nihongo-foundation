package quiz

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

// chatServer answers chat completions with status and body, and keeps the
// last request it got.
type chatServer struct {
	status int
	body   string
	path   string
	header http.Header
	got    struct {
		Model    string `json:"model"`
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
		ResponseFormat struct {
			Type string `json:"type"`
		} `json:"response_format"`
	}
}

func (c *chatServer) start(t *testing.T) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.path, c.header = r.Method+" "+r.URL.Path, r.Header
		raw, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(raw, &c.got); err != nil {
			t.Errorf("request body: %v", err)
		}
		w.WriteHeader(c.status)
		_, _ = io.WriteString(w, c.body)
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

// reply wraps content as a chat completions reply.
func reply(content string) string {
	quoted, _ := json.Marshal(content)
	return `{"choices":[{"message":{"role":"assistant","content":` + string(quoted) + `}}]}`
}

var amaiTargets = []Target{{Code: "v101-001", Type: TypeMeaning, Prompt: "あまい", Answer: "manis",
	Facts: Facts{Level: "n5", Word: "あまい", Reading: "あまい", PartOfSpeech: "い-adjective", Meaning: "manis"}}}

const amaiDrafts = `{"questions":[{"code":"v101-001","type":"meaning","prompt":"","answer":"",` +
	`"distractors":["asin","pahit","pedas"],"explanation":"Amai berarti manis."}]}`

var amaiWant = []Draft{{Code: "v101-001", Type: TypeMeaning, Distractors: []string{"asin", "pahit", "pedas"}, Explanation: "Amai berarti manis."}}

func TestChatPostsTheTargetsAndParsesTheDrafts(t *testing.T) {
	chat := &chatServer{status: http.StatusOK, body: reply(amaiDrafts)}
	url := chat.start(t)

	drafts, err := NewChat("OpenRouter", url+"/v1/", "openrouter/free", "sk-1", map[string]string{"X-OpenRouter-Title": "Nihongo"}).
		Generate(context.Background(), LangID, amaiTargets)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(drafts, amaiWant) {
		t.Errorf("drafts: got %+v\nwant %+v", drafts, amaiWant)
	}
	if chat.path != "POST /v1/chat/completions" || chat.header.Get("Authorization") != "Bearer sk-1" ||
		chat.header.Get("X-OpenRouter-Title") != "Nihongo" || chat.got.Model != "openrouter/free" || chat.got.ResponseFormat.Type != "json_object" {
		t.Errorf("request: %s, auth %q, title %q, model %q, format %q", chat.path, chat.header.Get("Authorization"),
			chat.header.Get("X-OpenRouter-Title"), chat.got.Model, chat.got.ResponseFormat.Type)
	}
	if len(chat.got.Messages) != 2 || chat.got.Messages[0].Role != "system" || chat.got.Messages[0].Content == "" || chat.got.Messages[1].Role != "user" {
		t.Fatalf("messages: %+v; want a system and a user message", chat.got.Messages)
	}
	wantUser := `{"lang":"id","targets":[{"code":"v101-001","type":"meaning","item":{"level":"n5","word":"あまい","reading":"あまい","part_of_speech":"い-adjective","meaning":"manis"}}]}`
	if chat.got.Messages[1].Content != wantUser {
		t.Errorf("user message:\ngot  %s\nwant %s", chat.got.Messages[1].Content, wantUser)
	}

	fenced := &chatServer{status: http.StatusOK, body: reply("```json\n" + amaiDrafts + "\n```")}
	if drafts, err := NewChat("OpenCode Zen", fenced.start(t), "m", "k", nil).Generate(context.Background(), LangID, amaiTargets); err != nil || !reflect.DeepEqual(drafts, amaiWant) {
		t.Errorf("fenced content: got %+v, %v; want the drafts", drafts, err)
	}
}

func TestChatErrorsOnAnUnusableReplyAndMarksWhichFallThrough(t *testing.T) {
	for _, tt := range []struct {
		name        string
		status      int
		body        string
		want        string
		fallThrough bool
	}{
		{"unauthorized", http.StatusUnauthorized, `{"error":{"message":"bad key"}}`, "Zen answered 401", false},
		{"bad request", http.StatusBadRequest, `{}`, "Zen answered 400", false},
		{"payment required", http.StatusPaymentRequired, `{}`, "Zen answered 402", true},
		{"rate limited", http.StatusTooManyRequests, `{}`, "Zen answered 429", true},
		{"server error", http.StatusBadGateway, `{}`, "Zen answered 502", true},
		{"no choice", http.StatusOK, `{"choices":[]}`, "no content", false},
		{"empty content", http.StatusOK, reply(""), "no content", false},
		{"content not JSON", http.StatusOK, reply("Here are your questions."), "not the expected JSON", false},
		{"content without questions", http.StatusOK, reply(`{"answers":[]}`), "not the expected JSON", false},
		{"reply not JSON", http.StatusOK, `<html>`, "reply", false},
	} {
		chat := &chatServer{status: tt.status, body: tt.body}
		_, err := NewChat("Zen", chat.start(t), "m", "k", nil).Generate(context.Background(), LangEN, amaiTargets)
		if err == nil || !strings.Contains(err.Error(), tt.want) || fallsThrough(err) != tt.fallThrough {
			t.Errorf("%s: error = %v, falls through %v; want one naming %q, falls through %v", tt.name, err, fallsThrough(err), tt.want, tt.fallThrough)
		}
	}

	srv := httptest.NewServer(http.NotFoundHandler())
	srv.Close()
	if _, err := NewChat("Zen", srv.URL, "m", "k", nil).Generate(context.Background(), LangEN, amaiTargets); err == nil || !fallsThrough(err) {
		t.Errorf("unreachable server: error = %v, want one that falls through", err)
	}
}

func TestChainAnswersTheFirstSuccessFallingThroughOnlyWhereMarked(t *testing.T) {
	asked := []string{}
	provider := func(name string, err error) fakeModel {
		return func(ctx context.Context, lang Lang, targets []Target) ([]Draft, error) {
			asked = append(asked, name)
			if err != nil {
				return nil, err
			}
			return unchanged(ctx, lang, targets)
		}
	}
	busy := fallThrough{errors.New("OpenRouter answered 429")}
	broken := errors.New("OpenRouter answered 401")

	drafts, err := Chain{provider("openrouter", busy), provider("zen", nil)}.Generate(context.Background(), LangEN, amaiTargets)
	if err != nil || len(drafts) != 1 || !reflect.DeepEqual(asked, []string{"openrouter", "zen"}) {
		t.Errorf("busy then ok: asked %v, %d drafts, %v; want both asked and the second's drafts", asked, len(drafts), err)
	}

	asked = []string{}
	_, err = Chain{provider("openrouter", broken), provider("zen", nil)}.Generate(context.Background(), LangEN, amaiTargets)
	if err == nil || !reflect.DeepEqual(asked, []string{"openrouter"}) {
		t.Errorf("broken: asked %v, %v; want only the first asked and its error", asked, err)
	}

	asked = []string{}
	_, err = Chain{provider("openrouter", busy), provider("zen", fallThrough{errors.New("Zen answered 503")})}.Generate(context.Background(), LangEN, amaiTargets)
	if err == nil || !strings.Contains(err.Error(), "OpenRouter answered 429") || !strings.Contains(err.Error(), "Zen answered 503") {
		t.Errorf("all busy: %v; want both failures joined", err)
	}
}
