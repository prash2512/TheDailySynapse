package api

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func (s *Server) handleSync(w http.ResponseWriter, r *http.Request) {
	if err := s.syncer.TriggerSync(r.Context(), 20); err != nil {
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

func (s *Server) handleDeleteFeed(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	if idStr == "" {
		http.Error(w, "feed id is required", http.StatusBadRequest)
		return
	}

	var id int64
	_, err := fmt.Sscanf(idStr, "%d", &id)
	if err != nil {
		http.Error(w, "invalid feed id", http.StatusBadRequest)
		return
	}

	if err := s.store.MarkFeedForDeletion(id); err != nil {
		http.Error(w, fmt.Sprintf("failed to mark feed for deletion: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte("Feed marked for deletion"))
}
