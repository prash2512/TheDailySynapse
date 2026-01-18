package judge

import (
	"context"
	"log/slog"
	"time"

	"dailysynapse/backend/internal/config"
	"dailysynapse/backend/internal/store"
	"dailysynapse/backend/pkg/judge"
	"dailysynapse/backend/pkg/retry"
)

type Worker struct {
	store  store.ArticleStore
	scorer judge.Scorer
	cfg    *config.Config
	logger *slog.Logger
}

func NewWorker(s store.ArticleStore, scorer judge.Scorer, cfg *config.Config, logger *slog.Logger) *Worker {
	return &Worker{
		store:  s,
		scorer: scorer,
		cfg:    cfg,
		logger: logger,
	}
}

func (w *Worker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.cfg.JudgeInterval)
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
		w.logger.Error("failed to fetch unscored articles", slog.String("error", err.Error()))
		return
	}

	if len(articles) == 0 {
		return
	}

	article := articles[0]
	w.logger.Info("scoring article", slog.Int64("id", article.ID), slog.String("title", article.Title))

	var result *judge.ScoreResult
	err = retry.Do(ctx, 3, func() error {
		var scoreErr error
		result, scoreErr = w.scorer.Score(ctx, article.Title, article.Content)
		return scoreErr
	})

	if err != nil {
		w.logger.Error("failed to score article",
			slog.Int64("id", article.ID),
			slog.String("error", err.Error()),
		)
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
		w.logger.Error("failed to save score",
			slog.Int64("id", article.ID),
			slog.String("error", err.Error()),
		)
	} else {
		w.logger.Info("scored article",
			slog.String("title", article.Title),
			slog.Int("score", result.TotalScore),
		)
	}
}
