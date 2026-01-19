# TheDailySynapse

A personal RSS aggregator that uses an LLM to score and rank technical articles, surfacing the highest-signal content from your favorite engineering blogs.

## Why

Too many RSS feeds. Too many articles. Not enough time.

TheDailySynapse acts as a "Staff Engineer" filter—it reads every article, scores it for technical depth, novelty, and actionability, and gives you a ranked "Daily 5" of what's actually worth reading.

**What gets high scores:**
- Deep dives into system internals
- Post-mortems with real lessons
- Novel architectures and trade-offs

**What gets filtered out:**
- Marketing fluff
- Basic tutorials
- News recaps without depth

## Quick Start

### Prerequisites

- Go 1.22+
- A [Google Gemini API key](https://makersuite.google.com/app/apikey) (free tier works)

### Run Locally

```bash
# Clone and build
git clone https://github.com/yourusername/TheDailySynapse.git
cd TheDailySynapse
make build

# Set your API key
export GEMINI_API_KEY=your-api-key-here

# Start the server
./bin/synapse
```

The server starts at `http://localhost:8080`.

### Run with Docker

```bash
# Build and run
export GEMINI_API_KEY=your-api-key-here
make docker-run

# Or manually
docker build -t dailysynapse .
docker run -d \
  -p 8080:8080 \
  -v dailysynapse-data:/app/data \
  -e GEMINI_API_KEY=$GEMINI_API_KEY \
  dailysynapse
```

## Usage

### Add Feeds

```bash
# Add a feed
curl -X POST http://localhost:8080/api/feeds \
  -H "Content-Type: application/json" \
  -d '{"url": "https://blog.cloudflare.com/rss/", "name": "Cloudflare Blog"}'

# Add more
curl -X POST http://localhost:8080/api/feeds \
  -H "Content-Type: application/json" \
  -d '{"url": "https://go.dev/blog/feed.atom", "name": "Go Blog"}'
```

### Get Your Daily 5

```bash
curl http://localhost:8080/api/daily
```

Returns the top 5 highest-scored articles:

```json
{
  "data": [
    {
      "ID": 9,
      "Title": "The FIPS 140-3 Go Cryptographic Module",
      "QualityRank": 87,
      "Summary": "Deep dive into Go's new FIPS-compliant crypto module...",
      "Justification": "Excellent technical depth on cryptographic compliance..."
    }
  ]
}
```

### Filter by Topic

```bash
# Get Go-related articles
curl "http://localhost:8080/api/articles?tags=Go"

# Get Security or Cryptography articles  
curl "http://localhost:8080/api/articles?tags=Security,Cryptography"

# See all available tags
curl http://localhost:8080/api/tags
```

### Read Full Article

```bash
curl http://localhost:8080/api/articles/9
```

Returns the full article content (extracted and cleaned, images embedded as base64).

## API Reference

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/health` | Health check |
| `GET` | `/ready` | Readiness (DB) check |
| `GET` | `/api/feeds` | List all feeds |
| `POST` | `/api/feeds` | Add a feed `{"url": "...", "name": "..."}` |
| `DELETE` | `/api/feeds/{id}` | Remove a feed |
| `POST` | `/api/sync` | Trigger manual sync |
| `GET` | `/api/daily?limit=5` | Top N scored articles |
| `GET` | `/api/articles?tags=Go,Perf&limit=20` | Filter by tags |
| `GET` | `/api/articles/{id}` | Full article with content |
| `GET` | `/api/tags` | All tags with counts |

## Configuration

Environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | `synapse.db` | SQLite database path |
| `GEMINI_API_KEY` | (required) | Google Gemini API key |
| `PORT` | `8080` | Server port |
| `LOG_LEVEL` | `info` | Log level (debug/info/warn/error) |
| `SYNC_INTERVAL` | `15m` | How often to check feeds |
| `SYNC_BATCH_SIZE` | `20` | Feeds per sync batch |
| `JUDGE_INTERVAL` | `4s` | Time between scoring requests |
| `RETENTION_DAYS` | `30` | Days to keep articles |

## How It Works

1. **Syncer** polls RSS feeds, extracts clean article content
2. **Judge** sends articles to Gemini with a "Principal Engineer" persona prompt
3. **Scoring** rates each article 0-100 based on:
   - Technical Depth (40%)
   - Novelty (30%)
   - Timelessness (30%)
4. **Tags** are auto-generated for filtering
5. **Daily 5** surfaces the highest-scored articles

## Project Structure

```
backend/
├── cmd/synapse/        # Application entrypoint
├── internal/
│   ├── api/            # HTTP handlers, middleware
│   ├── config/         # Configuration loading
│   ├── core/           # Domain models
│   ├── judge/          # LLM scoring worker
│   ├── logging/        # Structured logging
│   ├── store/          # Database access
│   └── syncer/         # RSS sync worker
├── pkg/
│   ├── judge/          # Gemini client
│   ├── readability/    # Content extraction
│   └── retry/          # Retry utilities
└── scripts/            # SQL migrations
```

## License

MIT
