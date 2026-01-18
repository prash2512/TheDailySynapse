package api

import (
	"database/sql"
	"net/http"

	"dailysynapse/backend/internal/store"
	"dailysynapse/backend/internal/syncer"
)

type Server struct {
	db     *sql.DB
	store  *store.Queries
	syncer *syncer.Syncer
}

func NewServer(db *sql.DB, s *syncer.Syncer) *Server {
	return &Server{
		db:     db,
		store:  store.NewQueries(db),
		syncer: s,
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /sync", s.handleSync)
	mux.HandleFunc("GET /feeds", s.handleGetFeeds)
	mux.HandleFunc("POST /feeds", s.handleCreateFeed)

	return mux
}
