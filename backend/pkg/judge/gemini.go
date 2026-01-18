package judge

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

type GeminiClient struct {
	client           *genai.Client
	model            *genai.GenerativeModel
	maxContentLength int
}

func NewGeminiClient(apiKey string, maxContentLength int) (*GeminiClient, error) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create gemini client: %w", err)
	}

	model := client.GenerativeModel("gemini-2.5-pro")
	model.SetTemperature(0.2)
	model.ResponseMIMEType = "application/json"

	return &GeminiClient{
		client:           client,
		model:            model,
		maxContentLength: maxContentLength,
	}, nil
}

func (g *GeminiClient) Score(ctx context.Context, title string, content string) (*ScoreResult, error) {
	if len(content) > g.maxContentLength {
		content = content[:g.maxContentLength]
	}

	prompt := fmt.Sprintf(`
You are a Principal Software Engineer at a top-tier tech company.
Evaluate the following technical article for its quality and relevance to senior engineers.

Title: %s

Content:
<content>
%s
</content>

Your Goal:
Filter out marketing fluff, basic tutorials, and news recaps. Identify "Deep Magic"—internals, trade-offs, and timeless engineering principles.

Scoring Rubric (0-10):
- Technical Depth: Does it explain HOW/WHY or just THAT? Code snippets? Internals?
- Novelty: New information or rehash?
- Timelessness: Will this matter in 5 years?

Output strictly in valid JSON format:
{
  "technical_depth": 0-10,
  "novelty": 0-10,
  "timelessness": 0-10,
  "total_score": 0-100 (Weighted: Depth*4 + Novelty*3 + Timelessness*3),
  "summary": "One sentence summary for a busy CTO",
  "justification": "Why did you give this score? Be critical.",
  "tags": ["Tag1", "Tag2"]
}
`, title, content)

	resp, err := g.model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("gemini api error: %w", err)
	}

	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil {
		return nil, fmt.Errorf("empty response from gemini")
	}

	var jsonStr string
	for _, part := range resp.Candidates[0].Content.Parts {
		if txt, ok := part.(genai.Text); ok {
			jsonStr += string(txt)
		}
	}

	jsonStr = strings.TrimPrefix(jsonStr, "```json")
	jsonStr = strings.TrimPrefix(jsonStr, "```")
	jsonStr = strings.TrimSuffix(jsonStr, "```")
	jsonStr = strings.TrimSpace(jsonStr)

	var result ScoreResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("failed to parse json response: %w. Raw: %s", err, jsonStr)
	}

	return &result, nil
}
