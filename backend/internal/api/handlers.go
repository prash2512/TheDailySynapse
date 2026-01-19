package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

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

func (s *Server) handleGetDaily(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 20
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	offsetStr := r.URL.Query().Get("offset")
	offset := 0
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	articles, total, err := s.store.GetTopArticles(r.Context(), limit, offset)
	if err != nil {
		Error(w, http.StatusInternalServerError, fmt.Sprintf("failed to fetch articles: %v", err))
		return
	}
	JSON(w, http.StatusOK, map[string]any{"articles": articles, "total": total, "limit": limit, "offset": offset})
}

func (s *Server) handleDismissArticle(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid article id")
		return
	}

	if err := s.store.DeleteArticle(r.Context(), id); err != nil {
		Error(w, http.StatusInternalServerError, fmt.Sprintf("failed to dismiss: %v", err))
		return
	}

	JSON(w, http.StatusOK, map[string]string{"message": "article dismissed"})
}

func (s *Server) handleGetArticle(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid article id")
		return
	}

	article, err := s.store.GetArticleByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, core.ErrNotFound) {
			Error(w, http.StatusNotFound, "article not found")
			return
		}
		Error(w, http.StatusInternalServerError, fmt.Sprintf("failed to fetch article: %v", err))
		return
	}

	JSON(w, http.StatusOK, article)
}

func (s *Server) handleGetArticles(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 20
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	tagsParam := r.URL.Query().Get("tags")
	var tags []string
	if tagsParam != "" {
		for _, t := range strings.Split(tagsParam, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				tags = append(tags, t)
			}
		}
	}

	articles, err := s.store.GetArticlesByTags(r.Context(), tags, limit)
	if err != nil {
		Error(w, http.StatusInternalServerError, fmt.Sprintf("failed to fetch articles: %v", err))
		return
	}
	JSON(w, http.StatusOK, articles)
}

func (s *Server) handleGetTags(w http.ResponseWriter, r *http.Request) {
	tags, err := s.store.GetAllTags(r.Context())
	if err != nil {
		Error(w, http.StatusInternalServerError, fmt.Sprintf("failed to fetch tags: %v", err))
		return
	}
	JSON(w, http.StatusOK, tags)
}

func (s *Server) handleMarkRead(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid article id")
		return
	}

	if err := s.store.MarkArticleRead(r.Context(), id); err != nil {
		Error(w, http.StatusInternalServerError, fmt.Sprintf("failed to mark read: %v", err))
		return
	}

	JSON(w, http.StatusOK, map[string]string{"message": "marked as read"})
}

func (s *Server) handleToggleSaved(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid article id")
		return
	}

	saved, err := s.store.ToggleArticleSaved(r.Context(), id)
	if err != nil {
		Error(w, http.StatusInternalServerError, fmt.Sprintf("failed to toggle saved: %v", err))
		return
	}

	JSON(w, http.StatusOK, map[string]bool{"saved": saved})
}

func (s *Server) handleGetSaved(w http.ResponseWriter, r *http.Request) {
	articles, err := s.store.GetSavedArticles(r.Context())
	if err != nil {
		Error(w, http.StatusInternalServerError, fmt.Sprintf("failed to fetch saved articles: %v", err))
		return
	}
	JSON(w, http.StatusOK, articles)
}
