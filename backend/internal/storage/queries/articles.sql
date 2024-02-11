-- https://docs.sqlc.dev/en/stable/reference/query-annotations.html

-- name: NewArticle :one
INSERT INTO articles (
    title, preface, content,
    origin, viewers_count, published_at
) VALUES (
    @title, @preface, @content,
    @origin, @viewers_count, @published_at
) RETURNING id;

-- name: AttachArticleImage :exec
INSERT INTO article_images (
    article_id, image_id, main
) VALUES (
    @article_id, @image_id, @main
);

-- name: Articles :many
SELECT * FROM articles LIMIT @sql_limit OFFSET @sql_offset;

-- name: GetArticleByID :one
SELECT * FROM articles where id = @id;

-- name: GetArticleIDByTitleAndOrigin :one
SELECT id FROM articles
WHERE (@title::text = '' OR title ILIKE '%' || @title || '%')
AND origin = @origin;

-- name: UpdateArticleStats :exec
UPDATE articles
SET
viewers_count = @viewers_count,
updated_at = @updated_at
WHERE id = @id;

-- name: NewImage :one
INSERT INTO images (
    url
) VALUES (
    @url
) RETURNING id;
