package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
	nats "github.com/nats-io/nats.go"
	"github.com/romashorodok/news-tracker/backend/internal/storage"
	"github.com/romashorodok/news-tracker/pkg/natsinfo"
	"go.uber.org/fx"
)

type DatabaseConfig struct {
	Username string
	Password string
	Database string
	Host     string
	Port     string
	Driver   string
}

func (dconf *DatabaseConfig) GetURI() string {
	return fmt.Sprintf("%s://%s:%s@%s:%s/%s",
		dconf.Driver,
		dconf.Username,
		dconf.Password,
		dconf.Host,
		dconf.Port,
		dconf.Database,
	)
}

func NewDatabaseConfig() *DatabaseConfig {
	return &DatabaseConfig{
		Driver:   "postgres",
		Username: "admin",
		Password: "admin",
		Host:     "postgres",
		Port:     "5432",
		Database: "postgres",
	}
}

func WithTransaction(db *sql.DB, fn func(queries *storage.Queries) error) (err error) {
	tx, err := db.Begin()
	if err != nil {
		return
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		} else if err != nil {
			err = tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	err = fn(storage.New(tx))
	return err
}

type NewDatabaseConnectionParams struct {
	fx.In
	Lifecycle fx.Lifecycle

	Config *DatabaseConfig
}

func NewDatabaseConnection(params NewDatabaseConnectionParams) (*sql.DB, error) {
	conn, err := sql.Open(params.Config.Driver, params.Config.GetURI()+"?sslmode=disable")
	if err != nil {
		return nil, err
	}
	params.Lifecycle.Append(fx.StopHook(conn.Close))
	return conn, nil
}

type ArticleService struct {
	db      *sql.DB
	queries *storage.Queries
}

func (s *ArticleService) GetArticleIDByTitleAndOrigin(ctx context.Context, params storage.GetArticleIDByTitleAndOriginParams) (int64, error) {
	return s.queries.GetArticleIDByTitleAndOrigin(ctx, params)
}

type NewArticleParams struct {
	Article           storage.NewArticleParams
	MainImageURL      string
	ContentImagesURLs []string
}

var (
	ErrArticleRequireMainImage = errors.New("Article require at least the main image")
	ErrUnableCreateArticle     = errors.New("unable create the article")
	ErrUnableCreateImage       = errors.New("unable create the image")
)

func (s *ArticleService) newArticleImage(ctx context.Context, articleID int64, url string, main bool) error {
	imageID, err := s.queries.NewImage(ctx, url)
	if err != nil {
		return ErrUnableCreateImage
	}

	if err = s.queries.AttachArticleImage(ctx, storage.AttachArticleImageParams{
		ArticleID: articleID,
		ImageID:   imageID,
		Main:      main,
	}); err != nil {
		return ErrUnableCreateImage
	}
	return nil
}

func (s *ArticleService) NewArticle(ctx context.Context, params NewArticleParams) (id int64, err error) {
	if params.MainImageURL == "" {
		return 0, ErrArticleRequireMainImage
	}

	err = WithTransaction(s.db, func(queries *storage.Queries) error {
		articleID, err := s.queries.NewArticle(ctx, params.Article)
		if err != nil {
			log.Printf("unable create the article. Err:%s", err)
			return ErrUnableCreateArticle
		}

		if err = s.newArticleImage(ctx, articleID, params.MainImageURL, true); err != nil {
			log.Printf("unable create the article image. Err:%s", err)
			return err
		}

		for _, imageURL := range params.ContentImagesURLs {
			if err = s.newArticleImage(ctx, articleID, imageURL, false); err != nil {
				log.Printf("unable create the article image. Err:%s", err)
				return err
			}
		}

		id = articleID
		return nil
	})
	return id, err
}

func (s *ArticleService) UpdateArticleStats(ctx context.Context, params storage.UpdateArticleStatsParams) error {
	return s.queries.UpdateArticleStats(ctx, params)
}

type NewArticleServiceParams struct {
	fx.In

	DB *sql.DB
}

func NewArticleSerivce(params NewArticleServiceParams) *ArticleService {
	return &ArticleService{
		db:      params.DB,
		queries: storage.New(params.DB),
	}
}

type articleConsumerWorker struct {
	js             nats.JetStreamContext
	articleService *ArticleService
}

func (a *articleConsumerWorker) handler(ctx context.Context) func(msg *nats.Msg) {
	return func(msg *nats.Msg) {
		var article natsinfo.Article

		if err := article.Unmarshal(msg.Data); err != nil {
			log.Println("Unable deserialize %s article payload. Err:%s", msg.Subject, err)
			_ = msg.Ack()
		}

		articleID, err := a.articleService.GetArticleIDByTitleAndOrigin(ctx, storage.GetArticleIDByTitleAndOriginParams{
			Title:  article.Title,
			Origin: article.Origin,
		})
		if errors.Is(err, sql.ErrNoRows) {
			if _, err := a.articleService.NewArticle(ctx, NewArticleParams{
				Article: storage.NewArticleParams{
					Title:        article.Title,
					Preface:      article.Preface,
					Content:      article.Content,
					Origin:       article.Origin,
					ViewersCount: int32(article.ViewersCount),
					PublishedAt:  article.PublishedAt,
				},
				MainImageURL:      article.MainImage,
				ContentImagesURLs: article.ContentImages,
			}); err == nil {
				log.Printf("create the %+v", article)
				// _ = msg.Ack(opts ...nats.AckOpt)
				return
			}
		} else if err != nil {
			log.Printf("Unexpected database error for Title:%s Origin:%s. Err:%s", article.Title, article.Origin, err)
			return
		}

		if err = a.articleService.UpdateArticleStats(ctx, storage.UpdateArticleStatsParams{
			ViewersCount: int32(article.ViewersCount),
			UpdatedAt:    time.Now(),
			ID:           articleID,
		}); err != nil {
			log.Printf("Unable update article for Title:%s Origin:%s. Err:%s", article.Title, article.Origin, err)
			return
		}
		log.Printf("update the %+v", article)

		// msg.Ack(opts ...nats.AckOpt)
	}
}

func (a *articleConsumerWorker) start(ctx context.Context) {
	if _, err := natsinfo.CreateOrUpdateStream(a.js, natsinfo.ARTICLES_STREAM_CONFIG); err != nil {
		log.Panicf("unable set-up nats %s stream. Err:%s", natsinfo.ARTICLES_STREAM_CONFIG.Name, err)
		os.Exit(1)
	}

	queueGroup := "backend-articles-consumer"
	stream, subject, subOpts, config := natsinfo.ArticlesStream_NewArticleConsumerConfig(queueGroup)

	if _, err := natsinfo.CreateOrUpdateConsumer(a.js, stream, config); err != nil {
		log.Panicf("unable set-up nats %s consumer. Err:%s", queueGroup, err)
		os.Exit(1)
	}

	if _, err := a.js.QueueSubscribe(subject, queueGroup, a.handler(ctx), subOpts...); err != nil {
		log.Panicf("unable start nats %s consumer. Err:%s", queueGroup, err)
		os.Exit(1)
	}

	<-ctx.Done()
}

type NewArticleConsumerWorkerParams struct {
	fx.In

	JS             nats.JetStreamContext
	ArticleService *ArticleService
}

func StartArticleConsumerWorker(params NewArticleConsumerWorkerParams) {
	worker := &articleConsumerWorker{
		js:             params.JS,
		articleService: params.ArticleService,
	}
	worker.start(context.Background())
}

func main() {
	fx.New(
		fx.Provide(
			natsinfo.NewNatsConfig,
			natsinfo.NewNatsConnection,

			NewDatabaseConfig,
			NewDatabaseConnection,

			NewArticleSerivce,
		),
		fx.Invoke(StartArticleConsumerWorker),
	).Run()
}
