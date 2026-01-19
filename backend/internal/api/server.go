package api

import (
	"database/sql"
	"log/slog"
	"net/http"

	"dailysynapse/backend/internal/store"
	"dailysynapse/backend/internal/syncer"
)

type Server struct {
	db     *sql.DB
	store  store.Store
	syncer *syncer.Syncer
	logger *slog.Logger
}

func NewServer(db *sql.DB, s *syncer.Syncer, logger *slog.Logger) *Server {
	return &Server{
		db:     db,
		store:  store.NewQueries(db),
		syncer: s,
		logger: logger,
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /ready", s.handleReady)

	mux.HandleFunc("POST /api/sync", s.handleSync)
	mux.HandleFunc("GET /api/feeds", s.handleGetFeeds)
	mux.HandleFunc("POST /api/feeds", s.handleCreateFeed)
	mux.HandleFunc("DELETE /api/feeds/{id}", s.handleDeleteFeed)
	mux.HandleFunc("GET /api/daily", s.handleGetDaily)
	mux.HandleFunc("GET /api/articles", s.handleGetArticles)
	mux.HandleFunc("GET /api/articles/{id}", s.handleGetArticle)
	mux.HandleFunc("GET /api/tags", s.handleGetTags)

	return chain(mux,
		corsMiddleware,
		loggingMiddleware(s.logger),
		recoveryMiddleware(s.logger),
	)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	if err := s.db.PingContext(r.Context()); err != nil {
		Error(w, http.StatusServiceUnavailable, "database unavailable")
		return
	}
	JSON(w, http.StatusOK, map[string]string{"status": "ready"})
}
