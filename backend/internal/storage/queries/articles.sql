-- https://docs.sqlc.dev/en/stable/reference/query-annotations.html

-- name: NewArticle :one
INSERT INTO articles (
    title, preface, content,
    origin, viewers_count, published_at
) VALUES (
    @title, @preface, @content,
    @origin, @viewers_count, @published_at
) RETURNING id;

-- name: NewArticleImage :one
INSERT INTO article_images (
    article_id, url
) VALUES (
    @article_id, @url
) RETURNING id;

-- name: Articles :many
SELECT * FROM articles LIMIT @sql_limit OFFSET @sql_offset;

-- name: GetArticleByID :one
SELECT * FROM articles where id = @id;

