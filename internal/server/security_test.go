package server

import (
	"context"
	"crypto/tls"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/KrishRVH/boring-stack/internal/realtime"
)

func TestNewWiresSecurityHeaders(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	app := New("127.0.0.1:0", logger, nil, realtime.NewMemoryBus(), nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/healthz", nil)
	app.server.Handler.ServeHTTP(rec, req)

	assertHeader(t, rec.Result().Header, "Content-Security-Policy", contentSecurityPolicy)
	assertHeader(t, rec.Result().Header, "X-Content-Type-Options", "nosniff")
}

func TestSecurityHeadersAreSet(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	handler := (&App{}).securityHeaders(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	handler.ServeHTTP(rec, req)

	headers := rec.Result().Header
	assertHeader(t, headers, "Cross-Origin-Opener-Policy", "same-origin")
	assertHeader(t, headers, "Cross-Origin-Resource-Policy", "same-origin")
	assertHeader(t, headers, "Origin-Agent-Cluster", "?1")
	assertHeader(t, headers, "Referrer-Policy", "strict-origin-when-cross-origin")
	assertHeader(t, headers, "X-Content-Type-Options", "nosniff")
	assertHeader(t, headers, "X-Frame-Options", "DENY")
	assertHeader(t, headers, "X-XSS-Protection", "0")

	csp := headers.Get("Content-Security-Policy")
	for _, want := range []string{
		"default-src 'self'",
		"base-uri 'none'",
		"object-src 'none'",
		"frame-ancestors 'none'",
		"form-action 'self'",
		"connect-src 'self'",
		"script-src 'self' 'unsafe-eval'",
		"style-src 'self' 'unsafe-inline'",
		"worker-src 'none'",
	} {
		if !strings.Contains(csp, want) {
			t.Fatalf("Content-Security-Policy missing %q in %q", want, csp)
		}
	}

	if got := headers.Get("Strict-Transport-Security"); got != "" {
		t.Fatalf("Strict-Transport-Security on plain HTTP = %q, want empty", got)
	}
}

func TestSecurityHeadersSetHSTSForHTTPS(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*http.Request)
	}{
		{
			name: "tls",
			setup: func(r *http.Request) {
				r.TLS = &tls.ConnectionState{}
			},
		},
		{
			name: "x forwarded proto",
			setup: func(r *http.Request) {
				r.Header.Set("X-Forwarded-Proto", "https")
			},
		},
		{
			name: "forwarded",
			setup: func(r *http.Request) {
				r.Header.Set("Forwarded", "for=192.0.2.60; proto=https; host=example.com")
			},
		},
		{
			name: "forwarded quoted",
			setup: func(r *http.Request) {
				r.Header.Set("Forwarded", `for=192.0.2.60; proto="https"; host=example.com`)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
			tt.setup(req)
			handler := (&App{}).securityHeaders(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			}))

			handler.ServeHTTP(rec, req)

			assertHeader(t, rec.Result().Header, "Strict-Transport-Security", strictTransportSecurity)
		})
	}
}

func TestSecurityHeadersDoNotSetHSTSForForwardedNoise(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*http.Request)
	}{
		{
			name: "x forwarded proto first value is http",
			setup: func(r *http.Request) {
				r.Header.Set("X-Forwarded-Proto", "http, https")
			},
		},
		{
			name: "forwarded proto suffix",
			setup: func(r *http.Request) {
				r.Header.Set("Forwarded", "for=192.0.2.60; proto=httpsx; host=example.com")
			},
		},
		{
			name: "forwarded different parameter",
			setup: func(r *http.Request) {
				r.Header.Set("Forwarded", "for=192.0.2.60; xproto=https; host=example.com")
			},
		},
		{
			name: "forwarded first value is http",
			setup: func(r *http.Request) {
				r.Header.Set("Forwarded", "for=192.0.2.60; proto=http, for=192.0.2.61; proto=https")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
			tt.setup(req)
			handler := (&App{}).securityHeaders(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			}))

			handler.ServeHTTP(rec, req)

			if got := rec.Result().Header.Get("Strict-Transport-Security"); got != "" {
				t.Fatalf("Strict-Transport-Security = %q, want empty", got)
			}
		})
	}
}

func assertHeader(t *testing.T, headers http.Header, name string, want string) {
	t.Helper()
	if got := headers.Get(name); got != want {
		t.Fatalf("%s = %q, want %q", name, got, want)
	}
}
