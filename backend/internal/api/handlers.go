package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"dailysynapse/backend/internal/core"
)

func (s *Server) handleSync(w http.ResponseWriter, r *http.Request) {
	if err := s.syncer.TriggerSync(r.Context()); err != nil {
		Error(w, http.StatusInternalServerError, fmt.Sprintf("failed to trigger sync: %v", err))
		return
	}
	JSON(w, http.StatusOK, map[string]string{"message": "sync triggered"})
}

func (s *Server) handleGetFeeds(w http.ResponseWriter, r *http.Request) {
	feeds, err := s.store.GetAllFeeds(r.Context())
	if err != nil {
		Error(w, http.StatusInternalServerError, fmt.Sprintf("failed to fetch feeds: %v", err))
		return
	}
	JSON(w, http.StatusOK, feeds)
}

func (s *Server) handleCreateFeed(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL  string `json:"url"`
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.URL == "" {
		Error(w, http.StatusBadRequest, "url is required")
		return
	}

	feed, err := s.store.CreateFeed(r.Context(), req.URL, req.Name)
	if err != nil {
		Error(w, http.StatusInternalServerError, fmt.Sprintf("failed to create feed: %v", err))
		return
	}
	JSON(w, http.StatusCreated, feed)
}

func (s *Server) handleDeleteFeed(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	if idStr == "" {
		Error(w, http.StatusBadRequest, "feed id is required")
		return
	}

	var id int64
	_, err := fmt.Sscanf(idStr, "%d", &id)
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid feed id")
		return
	}

	if err := s.store.MarkFeedForDeletion(r.Context(), id); err != nil {
		if errors.Is(err, core.ErrNotFound) {
			Error(w, http.StatusNotFound, "feed not found")
			return
		}
		Error(w, http.StatusInternalServerError, fmt.Sprintf("failed to delete feed: %v", err))
		return
	}

	JSON(w, http.StatusAccepted, map[string]string{"message": "feed marked for deletion"})
}
