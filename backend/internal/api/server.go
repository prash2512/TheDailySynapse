package api

import (
	"database/sql"
	"embed"
	"io/fs"
	"log/slog"
	"net/http"

	"dailysynapse/backend/internal/store"
	"dailysynapse/backend/internal/syncer"
)

//go:embed static
var staticFS embed.FS

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

	staticContent, _ := fs.Sub(staticFS, "static")
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticContent))))

	mux.HandleFunc("GET /{$}", s.handleDailyPage)
	mux.HandleFunc("GET /read/{id}", s.handleReaderPage)
	mux.HandleFunc("GET /feeds", s.handleFeedsPage)
	mux.HandleFunc("POST /feeds", s.handleFeedsPage)

	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /ready", s.handleReady)

	mux.HandleFunc("POST /api/sync", s.handleSync)
	mux.HandleFunc("GET /api/feeds", s.handleGetFeeds)
	mux.HandleFunc("POST /api/feeds", s.handleCreateFeed)
	mux.HandleFunc("DELETE /api/feeds/{id}", s.handleDeleteFeed)
	mux.HandleFunc("GET /api/daily", s.handleGetDaily)
	mux.HandleFunc("GET /api/articles", s.handleGetArticles)
	mux.HandleFunc("GET /api/articles/{id}", s.handleGetArticle)
	mux.HandleFunc("POST /api/articles/{id}/read", s.handleMarkRead)
	mux.HandleFunc("POST /api/articles/{id}/unread", s.handleMarkUnread)
	mux.HandleFunc("POST /api/articles/{id}/save", s.handleToggleSaved)
	mux.HandleFunc("DELETE /api/articles/{id}", s.handleDismissArticle)
	mux.HandleFunc("GET /api/saved", s.handleGetSaved)
	mux.HandleFunc("GET /api/tags", s.handleGetTags)

	mux.HandleFunc("GET /saved", s.handleSavedPage)

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
