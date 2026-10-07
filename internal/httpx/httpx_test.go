package httpx

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func serve(h http.Handler, method, target string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, target, nil))
	return rec
}

func TestWriteJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteJSON(rec, http.StatusOK, map[string]string{"title": "お名前は？"})
	if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "application/json" {
		t.Errorf("status %d, content type %q", rec.Code, rec.Header().Get("Content-Type"))
	}
	if got, want := rec.Body.String(), `{"title":"お名前は？"}`+"\n"; got != want {
		t.Errorf("body %q, want %q", got, want)
	}
}

func TestWriteError(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteError(rec, http.StatusNotFound, "lesson_not_found", "Lesson was not found.")
	want := `{"error":{"code":"lesson_not_found","message":"Lesson was not found."}}` + "\n"
	if rec.Code != http.StatusNotFound || rec.Body.String() != want {
		t.Errorf("got %d %q, want 404 %q", rec.Code, rec.Body.String(), want)
	}
}

func TestInternalErrorHidesTheCause(t *testing.T) {
	rec := httptest.NewRecorder()
	InternalError(rec, "get lesson", errors.New("password authentication failed for user postgres"))
	want := `{"error":{"code":"internal_error","message":"Internal server error."}}` + "\n"
	if rec.Code != http.StatusInternalServerError || rec.Body.String() != want {
		t.Errorf("got %d %q, want 500 %q", rec.Code, rec.Body.String(), want)
	}
}

func TestRouterAnswersUnmatchedRequestsInJSON(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) { WriteJSON(w, http.StatusOK, "ok") })
	h := Router(mux)

	ok := serve(h, http.MethodGet, "/health")
	if ok.Code != http.StatusOK || ok.Body.String() != `"ok"`+"\n" {
		t.Errorf("matched route: got %d %q", ok.Code, ok.Body.String())
	}

	notFound := serve(h, http.MethodGet, "/nope")
	if want := `{"error":{"code":"not_found","message":"Route was not found."}}` + "\n"; notFound.Code != http.StatusNotFound || notFound.Body.String() != want {
		t.Errorf("unknown path: got %d %q, want 404 %q", notFound.Code, notFound.Body.String(), want)
	}

	wrongMethod := serve(h, http.MethodDelete, "/health")
	if want := `{"error":{"code":"method_not_allowed","message":"Method is not allowed on this route."}}` + "\n"; wrongMethod.Code != http.StatusMethodNotAllowed || wrongMethod.Body.String() != want {
		t.Errorf("wrong method: got %d %q, want 405 %q", wrongMethod.Code, wrongMethod.Body.String(), want)
	}
	if allow := wrongMethod.Header().Get("Allow"); allow != "GET, HEAD" {
		t.Errorf("Allow = %q, want GET, HEAD", allow)
	}

	unclean := serve(h, http.MethodGet, "//health")
	if unclean.Code != http.StatusTemporaryRedirect || unclean.Header().Get("Location") != "/health" {
		t.Errorf("unclean path: got %d to %q, want the mux's redirect to /health", unclean.Code, unclean.Header().Get("Location"))
	}
}

func TestDocs(t *testing.T) {
	mux := http.NewServeMux()
	Docs(mux, []byte("openapi: 3.1.0\n"))
	h := Router(mux)

	for _, target := range []string{"/docs", "/docs/", "/docs/index.html"} {
		page := serve(h, http.MethodGet, target)
		if page.Code != http.StatusOK || page.Header().Get("Content-Type") != "text/html; charset=utf-8" ||
			!strings.Contains(page.Body.String(), "url: '/docs/openapi.yaml'") {
			t.Errorf("%s: got %d %q, want the docs page", target, page.Code, page.Header().Get("Content-Type"))
		}
	}

	spec := serve(h, http.MethodGet, "/docs/openapi.yaml")
	if spec.Code != http.StatusOK || spec.Header().Get("Content-Type") != "application/yaml" || spec.Body.String() != "openapi: 3.1.0\n" {
		t.Errorf("/docs/openapi.yaml: got %d %q %q", spec.Code, spec.Header().Get("Content-Type"), spec.Body.String())
	}

	root := serve(h, http.MethodGet, "/")
	if root.Code != http.StatusFound || root.Header().Get("Location") != "/docs" {
		t.Errorf("/: got %d to %q, want 302 to /docs", root.Code, root.Header().Get("Location"))
	}
	if rec := serve(h, http.MethodGet, "/docs/other"); rec.Code != http.StatusNotFound {
		t.Errorf("/docs/other: got %d, want the JSON 404", rec.Code)
	}
}
