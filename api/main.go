package main

import (
	"context"
	"flag"
	"log"
	"net/http"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	var (
		addr = flag.String("addr", ":8080", "endpoint address")
		mgo  = flag.String("mongo", "mongodb://localhost:27017", "MongoDB address")
	)
	log.Println("Dialing mongo", *mgo)
	db, err := mongo.Connect(context.Background(), options.Client().ApplyURI(*mgo))
	if err != nil {
		log.Fatal("Failed to connect to mongo:", err)
	}
	defer db.Disconnect(context.Background())

	s := &Server{
		db: db,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/polls/", withCORS(withAPIKey(s.handlePolls)))
	log.Println("Starting server on", *addr)
	http.ListenAndServe(*addr, mux)
	log.Println("Stopping")
}

// Server is the API server
type Server struct {
	db *mongo.Client
}

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
