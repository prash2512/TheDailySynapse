package readability

import (
	"fmt"
	"time"

	"github.com/go-shiori/go-readability"
)

// Extract fetches the URL and returns the readable HTML content.
func Extract(url string) (string, error) {
	article, err := readability.FromURL(url, 30*time.Second)
	if err != nil {
		return "", fmt.Errorf("failed to extract content from %s: %w", url, err)
	}

	return article.Content, nil
}
