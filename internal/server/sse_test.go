package server

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/KrishRVH/boring-stack/internal/appmodel"
)

func TestWriteSSEFormatsNamedHTMLPayload(t *testing.T) {
	var b bytes.Buffer
	err := writeSSE(&b, "snapshot", "<section>one</section>\n<section>two</section>")
	if err != nil {
		t.Fatal(err)
	}

	want := "event: snapshot\n" +
		"data: <section>one</section>\n" +
		"data: <section>two</section>\n\n"
	if got := b.String(); got != want {
		t.Fatalf("unexpected SSE payload\nwant: %q\n got: %q", want, got)
	}
}

func TestWriteSSENormalizesCarriageReturns(t *testing.T) {
	var b bytes.Buffer
	err := writeSSE(&b, "snapshot", "one\rtwo\r\nthree")
	if err != nil {
		t.Fatal(err)
	}

	want := "event: snapshot\n" +
		"data: one\n" +
		"data: two\n" +
		"data: three\n\n"
	if got := b.String(); got != want {
		t.Fatalf("unexpected SSE payload\nwant: %q\n got: %q", want, got)
	}
}

func TestRoutesDoNotLetHomeHandleUnknownPaths(t *testing.T) {
	app := &App{mux: http.NewServeMux()}
	app.routes()

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/missing", nil)
	rec := httptest.NewRecorder()
	app.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET /missing status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestValidateTodoBody(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{name: "empty", body: "", want: "Todo body is required."},
		{name: "whitespace", body: " \t\n ", want: "Todo body is required."},
		{name: "exact max ascii", body: strings.Repeat("x", appmodel.MaxTodoBodyLength), want: ""},
		{name: "over max ascii", body: strings.Repeat("x", appmodel.MaxTodoBodyLength+1), want: "Todo body must be 280 characters or fewer."},
		{name: "exact max multibyte", body: strings.Repeat("界", appmodel.MaxTodoBodyLength), want: ""},
		{name: "over max multibyte", body: strings.Repeat("界", appmodel.MaxTodoBodyLength+1), want: "Todo body must be 280 characters or fewer."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validateTodoBody(tt.body); got != tt.want {
				t.Fatalf("validateTodoBody() = %q, want %q", got, tt.want)
			}
		})
	}
}
