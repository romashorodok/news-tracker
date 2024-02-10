-- +goose Up
-- +goose StatementBegin
CREATE TABLE articles (
    id BIGSERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    preface VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    origin VARCHAR(255) NOT NULL,
    viewers_count int NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    published_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE article_images (
    id BIGSERIAL PRIMARY KEY,
    article_id BIGINT NOT NULL,
    url VARCHAR(510) NOT NULL,
    FOREIGN KEY(article_id) REFERENCES articles(id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS articles CASCADE;
DROP TABLE IF EXISTS article_images;
-- +goose StatementEnd
