package worker

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"os"
	"time"

	nats "github.com/nats-io/nats.go"
	"github.com/romashorodok/news-tracker/backend/internal/service"
	"github.com/romashorodok/news-tracker/backend/internal/storage"
	"github.com/romashorodok/news-tracker/pkg/natsinfo"
	"go.uber.org/fx"
)

type articleConsumerWorker struct {
	js             nats.JetStreamContext
	articleService *service.ArticleService
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
			if articleID, err := a.articleService.NewArticle(ctx, service.NewArticleParams{
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
				log.Printf("Created article %d.", articleID)
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
		log.Printf("Update article %d stats.", articleID)
		// _ = msg.Ack(opts ...nats.AckOpt)
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

type StartArticleConsumerWorkerParams struct {
	fx.In

	JS             nats.JetStreamContext
	ArticleService *service.ArticleService
}

func StartArticleConsumerWorker(params StartArticleConsumerWorkerParams) {
	worker := &articleConsumerWorker{
		js:             params.JS,
		articleService: params.ArticleService,
	}
	go worker.start(context.Background())
}
