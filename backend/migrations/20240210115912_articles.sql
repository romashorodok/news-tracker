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

CREATE TABLE images (
    id BIGSERIAL PRIMARY KEY,
    url VARCHAR(510) NOT NULL
);

CREATE TABLE article_images (
    article_id BIGINT NOT NULL,
    image_id BIGINT NOT NULL,
    main BOOLEAN NOT NULL,

    FOREIGN KEY(article_id) REFERENCES articles(id) ON DELETE CASCADE,
    FOREIGN KEY(image_id) REFERENCES images(id) ON DELETE CASCADE,
    UNIQUE(article_id, image_id),
    UNIQUE(image_id)
);

CREATE FUNCTION delete_article_images()
RETURNS TRIGGER AS $$
BEGIN
    DELETE FROM images 
    WHERE id IN (
        SELECT DISTINCT image_id FROM article_images WHERE article_id = OLD.id
    );
  RETURN OLD;
END;
$$ LANGUAGE plpgsql;
 
CREATE TRIGGER before_delete_article
BEFORE DELETE ON articles
FOR EACH ROW
EXECUTE FUNCTION delete_article_images();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DO $$ 
BEGIN
    IF EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'before_delete_article' AND tgrelid = 'articles'::regclass) THEN
        DROP TRIGGER IF EXISTS before_delete_article ON articles;
    END IF;

    IF EXISTS (SELECT 1 FROM pg_proc WHERE proname = 'delete_article_images') THEN
        DROP FUNCTION IF EXISTS delete_article_images();
    END IF;
END $$;

DROP TABLE IF EXISTS articles CASCADE;
DROP TABLE IF EXISTS images CASCADE;
DROP TABLE IF EXISTS article_images;
-- +goose StatementEnd
