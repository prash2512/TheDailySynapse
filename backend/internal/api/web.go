package api

import (
	"context"
	"embed"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"dailysynapse/backend/internal/core"
)

//go:embed templates/*.html
var templatesFS embed.FS

var pageTemplates map[string]*template.Template

func init() {
	pageTemplates = make(map[string]*template.Template)

	pages := []string{"daily", "reader", "feeds", "saved"}
	for _, page := range pages {
		t := template.Must(template.ParseFS(templatesFS, "templates/base.html", "templates/"+page+".html"))
		pageTemplates[page] = t
	}
}

type ArticleView struct {
	core.Article
	ReadingTime   int
	FormattedDate string
	Tags          []string
	Content       template.HTML
}

func toArticleView(a core.Article, tags []string) ArticleView {
	wordCount := len(strings.Fields(a.Content))
	readingTime := wordCount / 200
	if readingTime < 1 {
		readingTime = 1
	}

	return ArticleView{
		Article:       a,
		ReadingTime:   readingTime,
		FormattedDate: a.PublishedAt.Format("January 2, 2006"),
		Tags:          tags,
		Content:       template.HTML(a.Content),
	}
}

func renderPage(w http.ResponseWriter, page string, data any) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return pageTemplates[page].ExecuteTemplate(w, "base.html", data)
}

func (s *Server) handleDailyPage(w http.ResponseWriter, r *http.Request) {
	pageStr := r.URL.Query().Get("page")
	page := 1
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	perPage := 20
	offset := (page - 1) * perPage

	articles, total, err := s.store.GetTopArticles(r.Context(), perPage, offset)
	if err != nil {
		s.logger.Error("failed to get articles", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	var views []ArticleView
	for _, a := range articles {
		tags, _ := s.store.GetArticleTags(r.Context(), a.ID)
		views = append(views, toArticleView(a, tags))
	}

	allTags, _ := s.store.GetAllTags(r.Context())

	totalPages := (total + perPage - 1) / perPage
	if totalPages < 1 {
		totalPages = 1
	}

	data := map[string]any{
		"Nav":        "daily",
		"Date":       time.Now().Format("Monday, January 2"),
		"Articles":   views,
		"Tags":       allTags,
		"Page":       page,
		"TotalPages": totalPages,
		"Total":      total,
		"HasPrev":    page > 1,
		"HasNext":    page < totalPages,
		"PrevPage":   page - 1,
		"NextPage":   page + 1,
	}

	if err := renderPage(w, "daily", data); err != nil {
		s.logger.Error("template error", "error", err)
	}
}

func (s *Server) handleReaderPage(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid article ID", http.StatusBadRequest)
		return
	}

	article, err := s.store.GetArticleByID(r.Context(), id)
	if err != nil {
		if err == core.ErrNotFound {
			http.Error(w, "Article not found", http.StatusNotFound)
			return
		}
		s.logger.Error("failed to get article", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	tags, _ := s.store.GetArticleTags(r.Context(), article.ID)
	view := toArticleView(*article, tags)

	data := map[string]any{
		"Title":   article.Title,
		"Article": view,
	}

	if err := renderPage(w, "reader", data); err != nil {
		s.logger.Error("template error", "error", err)
	}
}

func (s *Server) handleFeedsPage(w http.ResponseWriter, r *http.Request) {
	var message, messageType string

	if r.Method == http.MethodPost {
		url := strings.TrimSpace(r.FormValue("url"))
		if url != "" {
			_, err := s.store.CreateFeed(r.Context(), url, "")
			if err != nil {
				if err == core.ErrConflict {
					message = "Feed already exists"
					messageType = "error"
				} else {
					message = "Failed to add feed"
					messageType = "error"
				}
			} else {
				message = "Feed added successfully"
				messageType = "success"
				go s.syncer.TriggerSync(context.Background())
			}
		}
	}

	feeds, err := s.store.GetAllFeeds(r.Context())
	if err != nil {
		s.logger.Error("failed to get feeds", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"Nav":         "feeds",
		"Title":       "Feeds",
		"Feeds":       feeds,
		"Message":     message,
		"MessageType": messageType,
	}

	if err := renderPage(w, "feeds", data); err != nil {
		s.logger.Error("template error", "error", err)
	}
}

func (s *Server) handleSavedPage(w http.ResponseWriter, r *http.Request) {
	articles, err := s.store.GetSavedArticles(r.Context())
	if err != nil {
		s.logger.Error("failed to get saved articles", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	var views []ArticleView
	for _, a := range articles {
		tags, _ := s.store.GetArticleTags(r.Context(), a.ID)
		views = append(views, toArticleView(a, tags))
	}

	data := map[string]any{
		"Nav":      "saved",
		"Title":    "Saved",
		"Articles": views,
	}

	if err := renderPage(w, "saved", data); err != nil {
		s.logger.Error("template error", "error", err)
	}
}
