package readability

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-shiori/go-readability"
)

type Extractor interface {
	Extract(ctx context.Context, url string) (string, error)
}

type DefaultExtractor struct {
	timeout time.Duration
}

func NewExtractor(timeout time.Duration) *DefaultExtractor {
	return &DefaultExtractor{timeout: timeout}
}

func (e *DefaultExtractor) Extract(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	client := &http.Client{Timeout: e.timeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetching url: %w", err)
	}
	defer resp.Body.Close()

	article, err := readability.FromReader(resp.Body, req.URL)
	if err != nil {
		return "", fmt.Errorf("extracting content: %w", err)
	}

	return article.Content, nil
}
