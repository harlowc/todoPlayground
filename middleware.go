package main

import (
	"net/http"
	"net/url"
)

func securityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")

		if !sameOriginRequest(r) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func sameOriginRequest(r *http.Request) bool {
	if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
		return true
	}

	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}

	parsedOrigin, err := url.Parse(origin)
	if err != nil {
		return false
	}

	expectedScheme := "http"
	if r.TLS != nil {
		expectedScheme = "https"
	}
	return parsedOrigin.Scheme == expectedScheme && parsedOrigin.Host == r.Host
}
