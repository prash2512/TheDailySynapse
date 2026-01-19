package readability

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/go-shiori/go-readability"
)

type Extractor interface {
	Extract(ctx context.Context, url string) (string, error)
}

type DefaultExtractor struct {
	client  *http.Client
	timeout time.Duration
}

func NewExtractor(timeout time.Duration) *DefaultExtractor {
	return &DefaultExtractor{
		client:  &http.Client{Timeout: timeout},
		timeout: timeout,
	}
}

func (e *DefaultExtractor) Extract(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("User-Agent", "TheDailySynapse/1.0")

	resp, err := e.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetching url: %w", err)
	}
	defer resp.Body.Close()

	article, err := readability.FromReader(resp.Body, req.URL)
	if err != nil {
		return "", fmt.Errorf("extracting content: %w", err)
	}

	content := e.inlineImages(ctx, article.Content)
	return content, nil
}

var imgSrcRegex = regexp.MustCompile(`<img[^>]+src=["']([^"']+)["']`)

func (e *DefaultExtractor) inlineImages(ctx context.Context, html string) string {
	return imgSrcRegex.ReplaceAllStringFunc(html, func(match string) string {
		submatches := imgSrcRegex.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match
		}

		imgURL := submatches[1]
		if strings.HasPrefix(imgURL, "data:") {
			return match
		}

		dataURI, err := e.fetchAsDataURI(ctx, imgURL)
		if err != nil {
			return match
		}

		return strings.Replace(match, imgURL, dataURI, 1)
	})
}

func (e *DefaultExtractor) fetchAsDataURI(ctx context.Context, imgURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imgURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "TheDailySynapse/1.0")

	resp, err := e.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/jpeg"
	}
	contentType = strings.Split(contentType, ";")[0]

	if !strings.HasPrefix(contentType, "image/") {
		return "", fmt.Errorf("not an image: %s", contentType)
	}

	limitedReader := io.LimitReader(resp.Body, 5*1024*1024)
	data, err := io.ReadAll(limitedReader)
	if err != nil {
		return "", err
	}

	encoded := base64.StdEncoding.EncodeToString(data)
	return fmt.Sprintf("data:%s;base64,%s", contentType, encoded), nil
}
