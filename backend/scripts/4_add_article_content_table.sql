CREATE TABLE article_content (
    article_id INTEGER PRIMARY KEY REFERENCES articles(id) ON DELETE CASCADE,
    content TEXT,
    judge_model TEXT
);
