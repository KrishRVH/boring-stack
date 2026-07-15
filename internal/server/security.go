package server

import (
	"net/http"
	"strings"
)

const (
	contentSecurityPolicy = "default-src 'self'; " +
		"base-uri 'none'; " +
		"object-src 'none'; " +
		"frame-ancestors 'none'; " +
		"form-action 'self'; " +
		"connect-src 'self'; " +
		"script-src 'self' 'unsafe-eval'; " +
		"style-src 'self' 'unsafe-inline'; " +
		"img-src 'self' data:; " +
		"font-src 'self'; " +
		"media-src 'self'; " +
		"manifest-src 'self'; " +
		"worker-src 'none'"
	permissionsPolicy       = "accelerometer=(), camera=(), geolocation=(), gyroscope=(), magnetometer=(), microphone=(), payment=(), usb=()"
	strictTransportSecurity = "max-age=63072000; includeSubDomains"
)

func (a *App) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy", contentSecurityPolicy)
		h.Set("Cross-Origin-Opener-Policy", "same-origin")
		h.Set("Cross-Origin-Resource-Policy", "same-origin")
		h.Set("Origin-Agent-Cluster", "?1")
		h.Set("Permissions-Policy", permissionsPolicy)
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("X-XSS-Protection", "0")

		if requestIsHTTPS(r) {
			h.Set("Strict-Transport-Security", strictTransportSecurity)
		}

		next.ServeHTTP(w, r)
	})
}

func requestIsHTTPS(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	if strings.EqualFold(firstForwardedValue(r.Header.Get("X-Forwarded-Proto")), "https") {
		return true
	}
	return forwardedProtoIsHTTPS(r.Header.Get("Forwarded"))
}

func firstForwardedValue(value string) string {
	value, _, _ = strings.Cut(value, ",")
	return strings.TrimSpace(value)
}

func forwardedProtoIsHTTPS(value string) bool {
	for _, part := range strings.Split(firstForwardedValue(value), ";") {
		key, rawValue, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok || !strings.EqualFold(strings.TrimSpace(key), "proto") {
			continue
		}
		proto := strings.Trim(strings.TrimSpace(rawValue), `"`)
		if strings.EqualFold(proto, "https") {
			return true
		}
	}
	return false
}
