CREATE TABLE feeds (
    id INTEGER PRIMARY KEY,
    url TEXT UNIQUE NOT NULL,
    name TEXT,
    etag TEXT,
    last_synced_at DATETIME
);

CREATE TABLE articles (
    id INTEGER PRIMARY KEY,
    feed_id INTEGER REFERENCES feeds(id),
    title TEXT NOT NULL,
    url TEXT UNIQUE NOT NULL,
    published_at DATETIME,
    quality_rank INTEGER,
    summary TEXT,
    is_read BOOLEAN DEFAULT 0,
    read_later BOOLEAN DEFAULT 0
);

CREATE TABLE tags (
    id INTEGER PRIMARY KEY,
    name TEXT UNIQUE NOT NULL
);

CREATE TABLE article_tags (
    article_id INTEGER NOT NULL REFERENCES articles(id),
    tag_id INTEGER NOT NULL REFERENCES tags(id),
    PRIMARY KEY (article_id, tag_id)
);

CREATE INDEX idx_articles_quality_rank ON articles (quality_rank DESC);
CREATE INDEX idx_tags_name ON tags (name);
CREATE INDEX idx_article_tags_tag_id ON article_tags (tag_id);
