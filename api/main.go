package main

import (
	"context"
	"net/http"

	"go.mongodb.org/mongo-driver/mongo"
)

func main() {}

type contextKey struct {
	name string
}

var contextKeyAPIKey = &contextKey{"api-key"}

func APIKey(ctx context.Context) (string, bool) {
	key, ok := ctx.Value(contextKeyAPIKey).(string)
	return key, ok
}

func withAPIKey(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("X-API-Key")
		if !isValidAPIKey(apiKey) {
			respondErr(w, r, http.StatusUnauthorized, "invalid API key")
			return
		}
		ctx := context.WithValue(r.Context(), contextKeyAPIKey, apiKey)
		fn(w, r.WithContext(ctx))
	}
}

func isValidAPIKey(key string) bool {
	// For demonstration purposes, we accept a single hardcoded API key.
	return key == "abc123"
}

func withCORS(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Expose-Headers", "Location")
		fn(w, r)
	}
}
