package judge

import (
	"context"
	"log"
	"time"

	"dailysynapse/backend/internal/store"
	"dailysynapse/backend/pkg/judge"
)

type Worker struct {
	store  *store.Queries
	scorer judge.Scorer
}

func NewWorker(s *store.Queries, scorer judge.Scorer) *Worker {
	return &Worker{
		store:  s,
		scorer: scorer,
	}
}

func (w *Worker) Start(ctx context.Context) {
	ticker := time.NewTicker(4 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.processNextArticle(ctx)
		}
	}
}

func (w *Worker) processNextArticle(ctx context.Context) {
	articles, err := w.store.GetUnscoredArticles(ctx, 1)
	if err != nil {
		log.Printf("Judge: Failed to fetch unscored articles: %v", err)
		return
	}

	if len(articles) == 0 {
		return
	}

	article := articles[0]
	log.Printf("Judge: Scoring '%s'...", article.Title)

	result, err := w.scorer.Score(ctx, article.Title, article.Content)
	if err != nil {
		log.Printf("Judge: Failed to score article %d: %v", article.ID, err)
		return
	}

	err = w.store.UpdateArticleScore(ctx,
		article.ID,
		result.TotalScore,
		result.Summary,
		result.Justification,
		"gemini-2.5-pro",
		result.Tags,
	)
	if err != nil {
		log.Printf("Judge: Failed to save score for %d: %v", article.ID, err)
	} else {
		log.Printf("Judge: Scored '%s' -> %d/100", article.Title, result.TotalScore)
	}
}
