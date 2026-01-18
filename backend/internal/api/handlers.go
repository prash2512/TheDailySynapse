package api

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func (s *Server) handleSync(w http.ResponseWriter, r *http.Request) {
	if err := s.syncer.TriggerSync(r.Context()); err != nil {
		http.Error(w, fmt.Sprintf("failed to trigger sync: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Sync triggered successfully"))
}

func (s *Server) handleGetFeeds(w http.ResponseWriter, r *http.Request) {
	feeds, err := s.store.GetAllFeeds()
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to fetch feeds: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(feeds); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

func (s *Server) handleCreateFeed(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL  string `json:"url"`
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.URL == "" {
		http.Error(w, "url is required", http.StatusBadRequest)
		return
	}

	feed, err := s.store.CreateFeed(req.URL, req.Name)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to create feed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(feed)
}
