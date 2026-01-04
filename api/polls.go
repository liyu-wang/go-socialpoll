package main

import (
	"errors"
	"net/http"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type poll struct {
	ID      primitive.ObjectID `bson:"_id" json:"id"`
	Title   string             `bson:"title" json:"title"`
	Options []string           `bson:"options" json:"options"`
	Results map[string]int     `bson:"results,omitempty" json:"results,omitempty"`
	// only for demonstrating how we extract the api key from the context
	APIKey string `bson:"apikey" json:"apikey"`
}

func (s *Server) handlePolls(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleGetPolls(w, r)
		return
	case http.MethodPost:
		s.handleCreatePoll(w, r)
		return
	case http.MethodDelete:
		s.handleDeletePoll(w, r)
		return
	case http.MethodOptions:
		// CORS preflight
		w.Header().Set("Access-Control-Allow-Methods", "DELETE")
		respond(w, r, http.StatusOK, nil)
		return
	}
	// not found
	respondHTTPErr(w, r, http.StatusNotFound)
}

func (s *Server) handleGetPolls(w http.ResponseWriter, r *http.Request) {
	c := s.db.Database("ballots").Collection("polls")
	p := NewPath(r.URL.Path)
	if p.HasID() {
		// get specific poll
		objID, err := primitive.ObjectIDFromHex(p.ID)
		if err != nil {
			respondErr(w, r, http.StatusBadRequest, errors.New("invalid poll ID format"))
			return
		}
		var singlePoll *poll
		err = c.FindOne(r.Context(), bson.M{"_id": objID}).Decode(&singlePoll)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				respondErr(w, r, http.StatusNotFound, errors.New("poll not found"))
			} else {
				respondErr(w, r, http.StatusInternalServerError, err)
			}
			return
		}
		respond(w, r, http.StatusOK, &singlePoll)
	} else {
		// get all polls
		var result []*poll
		cursor, err := c.Find(r.Context(), bson.M{})
		if err != nil {
			respondErr(w, r, http.StatusInternalServerError, err)
			return
		}
		defer cursor.Close(r.Context())
		err = cursor.All(r.Context(), &result)
		if err != nil {
			respondErr(w, r, http.StatusInternalServerError, err)
			return
		}
		respond(w, r, http.StatusOK, &result)
	}
}

func (s *Server) handleCreatePoll(w http.ResponseWriter, r *http.Request) {
	c := s.db.Database("ballots").Collection("polls")
	var p poll
	if err := decodeBody(r, &p); err != nil {
		respondErr(w, r, http.StatusBadRequest, "failed to read poll from request", err)
		return
	}
	apikey, ok := APIKey(r.Context())
	if ok {
		p.APIKey = apikey
	}
	p.ID = primitive.NewObjectID()
	result, err := c.InsertOne(r.Context(), p)
	if err != nil {
		respondErr(w, r, http.StatusInternalServerError, "failed to create poll", err)
		return
	}

	w.Header().Set("Location", "polls/"+p.ID.Hex())
	respond(w, r, http.StatusCreated, result)
}

func (s *Server) handleDeletePoll(w http.ResponseWriter, r *http.Request) {
	c := s.db.Database("ballots").Collection("polls")
	p := NewPath(r.URL.Path)
	if !p.HasID() {
		respondErr(w, r, http.StatusBadRequest, "missing poll ID")
		return
	}
	objID, err := primitive.ObjectIDFromHex(p.ID)
	if err != nil {
		respondErr(w, r, http.StatusBadRequest, errors.New("invalid poll ID format"))
		return
	}
	result, err := c.DeleteOne(r.Context(), bson.M{"_id": objID})
	if err != nil {
		respondErr(w, r, http.StatusInternalServerError, "failed to delete poll", err)
		return
	}
	if result.DeletedCount == 0 {
		respondErr(w, r, http.StatusNotFound, "poll not found")
		return
	}
	respond(w, r, http.StatusOK, nil)
}
