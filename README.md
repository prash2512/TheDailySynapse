# TheDailySynapse

A personal RSS aggregator that uses an LLM to score and rank technical articles, surfacing the highest-signal content from your favorite engineering blogs. Features a modern web UI for browsing, filtering, and managing your curated article feed.

## Why

Too many RSS feeds. Too many articles. Not enough time.

TheDailySynapse acts as a "Staff Engineer" filter—it reads every article, scores it for technical depth, novelty, and timelessness, and gives you a ranked feed of what's actually worth reading.

**What gets high scores:**
- Deep dives into system internals
- Post-mortems with real lessons
- Novel architectures and trade-offs
- Timeless engineering principles

**What gets filtered out:**
- Marketing fluff
- Basic tutorials
- News recaps without depth
- Low-quality content (scores < 50)

## Features

- 🎯 **LLM-Powered Scoring**: Uses Gemini 2.5 Pro to score articles on technical depth, novelty, and timelessness
- 🌐 **Modern Web UI**: Beautiful, responsive interface for browsing articles
- 🔍 **Smart Filtering**: Filter by tags, search by title/summary
- 📌 **Article Management**: Mark as read/unread, save forever, dismiss articles
- 🏷️ **Auto-Tagging**: Automatic tag generation for easy topic filtering
- 📱 **Mobile-Friendly**: Responsive design works on all devices
- ⚡ **Fast & Efficient**: Uses RSS summaries for ranking (no full content download needed)

## Quick Start

### Prerequisites

- Go 1.22+
- A [Google Gemini API key](https://makersuite.google.com/app/apikey) (free tier works)

### Run Locally

```bash
# Clone the repository
git clone https://github.com/prash2512/TheDailySynapse.git
cd TheDailySynapse

# Build the application
make build

# Set your API key (or create a .env file)
export GEMINI_API_KEY=your-api-key-here

# Start the server
make run

# Or run directly
./bin/synapse
```

The web UI will be available at `http://localhost:8080`.

### Using .env File

Create a `.env` file in the project root:

```bash
GEMINI_API_KEY=your-api-key-here
DATABASE_URL=synapse.db
PORT=8080
LOG_LEVEL=info
```

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

## Web UI

TheDailySynapse includes a full-featured web interface:

### Main Feed (`/`)
- **Sticky Header**: Search bar and navigation always visible
- **Article Cards**: Display title, summary, feed name, date, quality score, and tags
- **Article Actions**:
  - ✅ **Done**: Mark article as read (moves to bottom)
  - 🔄 **Unread**: Mark article as unread (moves back to top)
  - 💾 **Save**: Save article forever (prevents auto-deletion)
  - ❌ **Dismiss**: Remove article from feed
- **Topic Filtering**: Click tags to filter articles by topic
- **Collapsible Tags**: Toggle tag visibility for cleaner UI
- **Pagination**: Navigate through pages of articles

### Reader Page (`/read/{id}`)
- **Interstitial Page**: Preview article before opening
- **Article Metadata**: Feed name, date, reading time, quality score
- **Action Buttons**: Same actions as main feed
- **Read on Original Site**: Opens article in new tab and redirects back

### Feeds Management (`/feeds`)
- Add new RSS feeds
- View all configured feeds
- Delete feeds

### Saved Articles (`/saved`)
- View all saved articles
- Articles marked as "Save forever" are preserved

## Usage

### Add Feeds via Web UI

1. Navigate to `http://localhost:8080/feeds`
2. Enter RSS feed URL and optional name
3. Click "Add Feed"

### Add Feeds via API

```bash
# Add a feed
curl -X POST http://localhost:8080/api/feeds \
  -H "Content-Type: application/json" \
  -d '{"url": "https://blog.cloudflare.com/rss/", "name": "Cloudflare Blog"}'

# Add more feeds
curl -X POST http://localhost:8080/api/feeds \
  -H "Content-Type: application/json" \
  -d '{"url": "https://go.dev/blog/feed.atom", "name": "Go Blog"}'
```

### Get Articles via API

```bash
# Get top articles (paginated)
curl "http://localhost:8080/api/daily?limit=20&offset=0"

# Filter by tags
curl "http://localhost:8080/api/articles?tags=Go,Performance&limit=20"

# Get specific article
curl http://localhost:8080/api/articles/123

# Get all tags
curl http://localhost:8080/api/tags
```

### Article Actions via API

```bash
# Mark article as read
curl -X POST http://localhost:8080/api/articles/123/read

# Mark article as unread
curl -X POST http://localhost:8080/api/articles/123/unread

# Toggle save status
curl -X POST http://localhost:8080/api/articles/123/save

# Dismiss article
curl -X DELETE http://localhost:8080/api/articles/123
```

## End-to-End Testing

### Manual E2E Test Flow

1. **Start the application**:
   ```bash
   make run
   ```

2. **Add a test feed**:
   ```bash
   curl -X POST http://localhost:8080/api/feeds \
     -H "Content-Type: application/json" \
     -d '{"url": "https://go.dev/blog/feed.atom", "name": "Go Blog"}'
   ```

3. **Trigger a sync**:
   ```bash
   curl -X POST http://localhost:8080/api/sync
   ```

4. **Wait for articles to be fetched** (check logs for sync completion)

5. **Wait for articles to be scored** (check logs for "scored article" messages)

6. **Verify articles appear in UI**:
   - Open `http://localhost:8080` in browser
   - Verify articles are displayed
   - Check that quality scores are shown

7. **Test article actions**:
   - Click "Done" on an article → should disappear or move to bottom
   - Click "Save" on an article → icon should fill
   - Click a tag → should filter articles
   - Use search bar → should filter by title/summary

8. **Test unread feature**:
   - Mark an article as read
   - Scroll down to find it (or refresh page)
   - Click "Unread" → article should move back to top

9. **Test reader page**:
   - Click an article title
   - Verify reader page shows article details
   - Click "Read on Original Site" → should open in new tab

10. **Verify saved articles**:
    - Navigate to `/saved`
    - Verify saved articles are listed

### Automated E2E Test Script

Create a test script `test-e2e.sh`:

```bash
#!/bin/bash
set -e

BASE_URL="http://localhost:8080"

echo "🧪 Starting E2E Tests..."

# Health check
echo "✓ Checking health..."
curl -f "$BASE_URL/health" > /dev/null

# Add feed
echo "✓ Adding test feed..."
FEED_ID=$(curl -s -X POST "$BASE_URL/api/feeds" \
  -H "Content-Type: application/json" \
  -d '{"url": "https://go.dev/blog/feed.atom", "name": "Go Blog"}' | jq -r '.id')

# Trigger sync
echo "✓ Triggering sync..."
curl -f -X POST "$BASE_URL/api/sync" > /dev/null

# Wait for articles
echo "⏳ Waiting for articles to be fetched and scored..."
sleep 30

# Check articles
echo "✓ Checking articles..."
ARTICLES=$(curl -s "$BASE_URL/api/daily?limit=5")
ARTICLE_COUNT=$(echo "$ARTICLES" | jq '.articles | length')

if [ "$ARTICLE_COUNT" -gt 0 ]; then
  echo "✅ Found $ARTICLE_COUNT articles"
  ARTICLE_ID=$(echo "$ARTICLES" | jq -r '.articles[0].id')
  
  # Test mark as read
  echo "✓ Testing mark as read..."
  curl -f -X POST "$BASE_URL/api/articles/$ARTICLE_ID/read" > /dev/null
  
  # Test mark as unread
  echo "✓ Testing mark as unread..."
  curl -f -X POST "$BASE_URL/api/articles/$ARTICLE_ID/unread" > /dev/null
  
  # Test save
  echo "✓ Testing save..."
  curl -f -X POST "$BASE_URL/api/articles/$ARTICLE_ID/save" > /dev/null
  
  echo "✅ All E2E tests passed!"
else
  echo "❌ No articles found. Check logs and API key."
  exit 1
fi
```

Run with: `chmod +x test-e2e.sh && ./test-e2e.sh`

## API Reference

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/` | Web UI - Main feed |
| `GET` | `/read/{id}` | Web UI - Reader page |
| `GET` | `/feeds` | Web UI - Feed management |
| `GET` | `/saved` | Web UI - Saved articles |
| `GET` | `/health` | Health check |
| `GET` | `/ready` | Readiness (DB) check |
| `GET` | `/api/feeds` | List all feeds |
| `POST` | `/api/feeds` | Add a feed `{"url": "...", "name": "..."}` |
| `DELETE` | `/api/feeds/{id}` | Remove a feed |
| `POST` | `/api/sync` | Trigger manual sync |
| `GET` | `/api/daily?limit=20&offset=0` | Top N scored articles (paginated) |
| `GET` | `/api/articles?tags=Go,Perf&limit=20` | Filter by tags |
| `GET` | `/api/articles/{id}` | Get article details |
| `POST` | `/api/articles/{id}/read` | Mark article as read |
| `POST` | `/api/articles/{id}/unread` | Mark article as unread |
| `POST` | `/api/articles/{id}/save` | Toggle save status |
| `DELETE` | `/api/articles/{id}` | Dismiss article |
| `GET` | `/api/tags` | All tags with counts |
| `GET` | `/api/saved` | All saved articles |

## Configuration

Environment variables (can be set in `.env` file):

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | `synapse.db` | SQLite database path |
| `GEMINI_API_KEY` | (required) | Google Gemini API key |
| `PORT` | `8080` | Server port |
| `LOG_LEVEL` | `info` | Log level (debug/info/warn/error) |
| `SYNC_INTERVAL` | `15m` | How often to check feeds |
| `SYNC_BATCH_SIZE` | `20` | Feeds per sync batch |
| `SYNC_WORKERS` | `5` | Number of sync workers |
| `JUDGE_INTERVAL` | `6s` | Time between scoring requests |
| `ARTICLE_HORIZON_DAYS` | `120` | Days back to fetch articles (4 months) |
| `RETENTION_DAYS` | `30` | Days to keep articles before auto-deletion |
| `MAX_CONTENT_LENGTH` | `20000` | Max characters for article summary |
| `HTTP_TIMEOUT` | `10s` | HTTP request timeout |

## How It Works

1. **Syncer** polls RSS feeds every 15 minutes, extracts article metadata and summaries
2. **Judge Worker** sends article summaries to Gemini 2.5 Pro with a "Principal Engineer" persona prompt
3. **Scoring** rates each article 0-100 based on:
   - Technical Depth (40% weight)
   - Novelty (30% weight)
   - Timelessness (30% weight)
4. **Auto-Tagging** generates tags for filtering (e.g., "Go", "Kubernetes", "Performance")
5. **Ranking** orders articles by read status, then quality score, then date
6. **Auto-Deletion** removes articles with scores < 50 after processing

### Architecture Highlights

- **Summary-Only Ranking**: Uses RSS feed summaries for scoring (no full content download)
- **Rate Limiting**: Built-in retry logic with exponential backoff for API limits
- **SQLite WAL Mode**: Enables concurrent reads/writes without locking
- **Background Workers**: Async feed syncing and article scoring
- **Structured Logging**: JSON logs for easy parsing and monitoring

## Project Structure

```
backend/
├── cmd/synapse/        # Application entrypoint
├── internal/
│   ├── api/            # HTTP handlers, middleware, web UI templates
│   │   ├── templates/  # HTML templates (daily, reader, feeds, saved)
│   │   └── static/      # CSS styles
│   ├── config/         # Configuration loading
│   ├── core/           # Domain models and errors
│   ├── judge/          # LLM scoring worker
│   ├── logging/        # Structured logging
│   ├── store/          # Database access layer
│   └── syncer/         # RSS sync worker
├── pkg/
│   ├── judge/          # Gemini client and scoring logic
│   ├── readability/    # Content extraction (legacy, not used)
│   └── retry/          # Retry utilities with rate limit handling
└── scripts/             # SQL migrations
```

## UI Screenshots & Features

### Main Feed
- Clean, card-based layout
- Sticky header with search
- Collapsible topic tags
- Article quality scores displayed prominently
- One-click actions (read/unread, save, dismiss)

### Reader Page
- Interstitial preview before opening external site
- Article metadata and summary
- Quick actions without leaving page

### Feed Management
- Simple form to add RSS feeds
- List view of all configured feeds
- Easy deletion

## Development

### Build

```bash
make build
```

### Run Tests

```bash
make test
```

### Format Code

```bash
make fmt
```

### Clean Build Artifacts

```bash
make clean
```

## Troubleshooting

### Articles Not Appearing

1. Check that feeds are syncing: Look for "sync" logs
2. Verify API key is set: `echo $GEMINI_API_KEY`
3. Check for rate limiting: Look for "rate limit hit" warnings
4. Verify articles are being scored: Look for "scored article" logs

### Rate Limiting Issues

- The app uses `gemini-2.5-pro` by default
- Judge interval is set to 6 seconds to avoid rate limits
- If you hit limits, increase `JUDGE_INTERVAL` in config
- Free tier API keys have lower limits

### Database Issues

- Database file: `synapse.db` (SQLite)
- If corrupted, delete and restart (feeds will need to be re-added)
- WAL files (`synapse.db-wal`, `synapse.db-shm`) are normal and safe

## License

MIT

## Contributing

Contributions welcome! Please open an issue or submit a PR.
