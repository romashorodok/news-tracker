-- https://docs.sqlc.dev/en/stable/reference/query-annotations.html
-- https://github.com/sqlc-dev/sqlc/issues/1062#issuecomment-869770485
-- https://docs.sqlc.dev/en/stable/howto/named_parameters.html#nullable-parameters

-- name: NewArticle :one
INSERT INTO articles (
    title, preface, content,
    origin, viewers_count, published_at
) VALUES (
    @title, @preface, @content,
    @origin, @viewers_count, @published_at
) RETURNING id;

-- name: Articles :many
SELECT
    articles.*,
    array_to_json(array_agg(row_to_json(images))) AS images
FROM articles
LEFT JOIN (
    SELECT DISTINCT ON (ai.image_id)
        ai.image_id,
        ai.main,
        i.url,
        ai.article_id
    FROM article_images ai
    JOIN images i ON ai.image_id = i.id
) AS images ON articles.id = images.article_id
WHERE
(
    articles.published_at >=
        COALESCE(sqlc.narg('start_date'), @start_date_default)::timestamp
    AND
    articles.published_at <= COALESCE(sqlc.narg('end_date'), NOW())::timestamp
)
AND
(
    CAST(ARRAY_TO_JSON(sqlc.slice('lexems')::text[]) AS VARCHAR) IN ('[null]', '[""]')  OR
    to_tsvector(articles.title || ' ' || articles.content || ' ' || articles.preface)
    @@ to_tsquery(ARRAY_TO_STRING(sqlc.slice('lexems'), ' & '))
)
GROUP BY articles.id
ORDER BY
    CASE WHEN @article_sorting::text = 'newest' THEN articles.published_at END DESC,
    CASE WHEN @article_sorting::text = 'oldest' THEN articles.published_at END ASC
LIMIT @page_size::bigint
OFFSET @page::bigint;

-- name: GetArticleByID :one
SELECT *, (
    SELECT
        array_to_json(array_agg(row_to_json(images))) AS json_array
    FROM (
        SELECT images.url, article_images.main
        FROM images
        JOIN (
            SELECT DISTINCT main, image_id
            FROM article_images
            WHERE article_id = @id
        ) AS article_images
        ON images.id = article_images.image_id
    ) as images
) as images
FROM articles
WHERE articles.id = @id;

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

-- name: AttachArticleImage :exec
INSERT INTO article_images (
    article_id, image_id, main
) VALUES (
    @article_id, @image_id, @main
);

-- name: GetArticleCount :one
SELECT COUNT(*)
FROM articles
WHERE
(
    articles.published_at >=
        COALESCE(sqlc.narg('start_date'), @start_date_default)::timestamp
    AND
    articles.published_at <= COALESCE(sqlc.narg('end_date'), NOW())::timestamp
)
AND
(
    CAST(ARRAY_TO_JSON(sqlc.slice('lexems')::text[]) AS VARCHAR) IN ('[null]', '[""]')  OR
    to_tsvector(articles.title || ' ' || articles.content || ' ' || articles.preface)
    @@ to_tsquery(ARRAY_TO_STRING(sqlc.slice('lexems'), ' & '))
);
