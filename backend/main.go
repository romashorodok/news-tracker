package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	chi "github.com/go-chi/chi/v5"
	_ "github.com/lib/pq"
	nats "github.com/nats-io/nats.go"
	"github.com/romashorodok/news-tracker/backend/internal/handler/v1"
	"github.com/romashorodok/news-tracker/backend/internal/service"
	"github.com/romashorodok/news-tracker/backend/internal/storage"
	"github.com/romashorodok/news-tracker/pkg/envutils"
	"github.com/romashorodok/news-tracker/pkg/httputils"
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
			if _, err := a.articleService.NewArticle(ctx, service.NewArticleParams{
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

type HttpServerConfig struct {
	Port string
	Host string
}

func (h *HttpServerConfig) GetAddr() string {
	return net.JoinHostPort(h.Host, h.Port)
}

func NewHttpServerConfig() *HttpServerConfig {
	return &HttpServerConfig{
		Host: envutils.Env("HTTP_HOST", ""),
		Port: envutils.Env("HTTP_PORT", "8080"),
	}
}

type StartHttpServerParams struct {
	fx.In

	Lifecycle fx.Lifecycle
	Config    *HttpServerConfig
	Handlers  []httputils.Handler `group:"http.handler"`
}

func StartHttpServer(params StartHttpServerParams) {
	router := chi.NewMux()

	server := &http.Server{
		Addr:    params.Config.GetAddr(),
		Handler: router,
	}

	router.Use(func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Content-Type", "application/json")

			handler.ServeHTTP(w, r)
		})
	})

	for _, handler := range params.Handlers {
		handler.OnRouter(router)
	}

	li, err := net.Listen("tcp", server.Addr)
	if err != nil {
		log.Panicf("Unable start http server. Err:%s", err)
		os.Exit(1)
	}

	params.Lifecycle.Append(fx.StopHook(func(ctx context.Context) error {
		return server.Shutdown(ctx)
	}))

	go server.Serve(li)
}

const groupHandler = `group:"http.handler"`

type ParamsNewNatsArticleKeyValue struct {
	fx.In

	Lifecycle fx.Lifecycle
	JS        nats.JetStreamContext
}

func NewNatsArticleKeyValue(params ParamsNewNatsArticleKeyValue) (nats.KeyValue, error) {
	return natsinfo.CreateOrAttachKeyValue(params.JS, &natsinfo.ARTICLE_COUNT_KEY_VALUE_CONFIG)
}

func main() {
	fx.New(
		fx.Provide(
			natsinfo.NewNatsConfig,
			natsinfo.NewNatsConnection,
			NewNatsArticleKeyValue,

			NewDatabaseConfig,
			NewDatabaseConnection,

			service.NewArticleSerivce,
			NewHttpServerConfig,

			httputils.AsHandler(groupHandler, handler.NewArticleHandler),
		),
		// fx.Invoke(StartArticleConsumerWorker),
		fx.Invoke(StartHttpServer),
	).Run()
}
