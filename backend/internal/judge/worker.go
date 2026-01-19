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
	err = retry.Do(ctx, 5, func() error {
		var scoreErr error
		result, scoreErr = w.scorer.Score(ctx, article.Title, article.Summary)
		return scoreErr
	})

	if err != nil {
		if retry.IsRateLimitError(err) {
			w.logger.Warn("rate limit hit, will retry later",
				slog.Int64("id", article.ID),
				slog.String("title", article.Title),
			)
		} else {
			w.logger.Error("failed to score article",
				slog.Int64("id", article.ID),
				slog.String("error", err.Error()),
			)
		}
		return
	}

	if result.TotalScore < 50 {
		if err := w.store.DeleteArticle(ctx, article.ID); err != nil {
			w.logger.Error("failed to delete low-score article",
				slog.Int64("id", article.ID),
				slog.String("error", err.Error()),
			)
		} else {
			w.logger.Info("deleted low-score article",
				slog.String("title", article.Title),
				slog.Int("score", result.TotalScore),
			)
		}
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
