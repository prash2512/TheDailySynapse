package judge

import (
	"context"
)

// ScoreResult represents the structured output from the LLM.
type ScoreResult struct {
	TechnicalDepth int      `json:"technical_depth"`
	Novelty        int      `json:"novelty"`
	Timelessness   int      `json:"timelessness"`
	TotalScore     int      `json:"total_score"`
	Summary        string   `json:"summary"`
	Justification  string   `json:"justification"`
	Tags           []string `json:"tags"`
}

// Scorer is the interface that any LLM provider must implement.
type Scorer interface {
	Score(ctx context.Context, title string, content string) (*ScoreResult, error)
}
