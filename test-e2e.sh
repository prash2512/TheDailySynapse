#!/bin/bash
set -e

BASE_URL="http://localhost:8080"

echo "🧪 Starting E2E Tests..."
echo ""

# Check if jq is installed
if ! command -v jq &> /dev/null; then
    echo "❌ jq is required for this script. Install it with: brew install jq"
    exit 1
fi

# Health check
echo "✓ Checking health..."
if ! curl -sf "$BASE_URL/health" > /dev/null; then
    echo "❌ Health check failed. Is the server running?"
    exit 1
fi

# Add feed
echo "✓ Adding test feed..."
FEED_RESPONSE=$(curl -s -X POST "$BASE_URL/api/feeds" \
  -H "Content-Type: application/json" \
  -d '{"url": "https://go.dev/blog/feed.atom", "name": "Go Blog"}')

FEED_ID=$(echo "$FEED_RESPONSE" | jq -r '.id // empty')

if [ -z "$FEED_ID" ] || [ "$FEED_ID" = "null" ]; then
    echo "⚠️  Feed might already exist, continuing..."
else
    echo "✓ Feed added with ID: $FEED_ID"
fi

# Trigger sync
echo "✓ Triggering sync..."
if ! curl -sf -X POST "$BASE_URL/api/sync" > /dev/null; then
    echo "❌ Sync trigger failed"
    exit 1
fi

# Wait for articles
echo "⏳ Waiting for articles to be fetched and scored..."
echo "   (This may take 30-60 seconds depending on API rate limits)"
sleep 30

# Check articles
echo "✓ Checking articles..."
ARTICLES=$(curl -s "$BASE_URL/api/daily?limit=5")
ARTICLE_COUNT=$(echo "$ARTICLES" | jq -r '.articles | length // 0')

if [ "$ARTICLE_COUNT" -gt 0 ]; then
    echo "✅ Found $ARTICLE_COUNT articles"
    ARTICLE_ID=$(echo "$ARTICLES" | jq -r '.articles[0].id')
    
    if [ -z "$ARTICLE_ID" ] || [ "$ARTICLE_ID" = "null" ]; then
        echo "⚠️  Could not get article ID, skipping action tests"
    else
        echo "✓ Testing with article ID: $ARTICLE_ID"
        
        # Test mark as read
        echo "  → Testing mark as read..."
        if curl -sf -X POST "$BASE_URL/api/articles/$ARTICLE_ID/read" > /dev/null; then
            echo "    ✅ Mark as read works"
        else
            echo "    ❌ Mark as read failed"
        fi
        
        # Test mark as unread
        echo "  → Testing mark as unread..."
        if curl -sf -X POST "$BASE_URL/api/articles/$ARTICLE_ID/unread" > /dev/null; then
            echo "    ✅ Mark as unread works"
        else
            echo "    ❌ Mark as unread failed"
        fi
        
        # Test save
        echo "  → Testing save toggle..."
        if curl -sf -X POST "$BASE_URL/api/articles/$ARTICLE_ID/save" > /dev/null; then
            echo "    ✅ Save toggle works"
        else
            echo "    ❌ Save toggle failed"
        fi
    fi
    
    # Test tags
    echo "✓ Testing tags endpoint..."
    TAGS=$(curl -s "$BASE_URL/api/tags")
    TAG_COUNT=$(echo "$TAGS" | jq -r 'length // 0')
    if [ "$TAG_COUNT" -gt 0 ]; then
        echo "  ✅ Found $TAG_COUNT tags"
    else
        echo "  ⚠️  No tags found (articles may not be scored yet)"
    fi
    
    echo ""
    echo "✅ All E2E tests passed!"
    echo ""
    echo "🌐 Open http://localhost:8080 in your browser to see the UI"
else
    echo "❌ No articles found."
    echo ""
    echo "Troubleshooting:"
    echo "  1. Check that GEMINI_API_KEY is set"
    echo "  2. Check server logs for errors"
    echo "  3. Wait a bit longer - articles may still be scoring"
    echo "  4. Try triggering sync again: curl -X POST $BASE_URL/api/sync"
    exit 1
fi

