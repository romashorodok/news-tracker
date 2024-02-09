package main

import (
	"context"
	"fmt"
	"log"
	"time"

	nats "github.com/nats-io/nats.go"
	"github.com/romashorodok/news-tracker/pkg/natsinfo"
	"github.com/romashorodok/news-tracker/worker/internal/prebuiltemplate"
	"go.uber.org/fx"
)

type NatsConfig struct {
	Port string
	Host string
}

func (c *NatsConfig) GetURL() string {
	if c.Host == "" || c.Port == "" {
		return nats.DefaultURL
	}
	return fmt.Sprintf("nats://%s:%s", c.Host, c.Port)
}

func NewNatsConfig() *NatsConfig {
	return &NatsConfig{}
}

type NewNatsConnectionResult struct {
	fx.Out

	Conn *nats.Conn
	JS   nats.JetStreamContext
}

func NewNatsConnection(config *NatsConfig) (NewNatsConnectionResult, error) {
	conn, err := nats.Connect(config.GetURL(),
		nats.Timeout(time.Second*5),
		nats.RetryOnFailedConnect(true),
	)
	if err != nil {
		return NewNatsConnectionResult{}, err
	}

	js, err := conn.JetStream()
	if err != nil {
		return NewNatsConnectionResult{}, err
	}

	if _, err = natsinfo.CreateStreamIfNotExists(js, natsinfo.ARTICLES_STREAM_CONFIG); err != nil {
		return NewNatsConnectionResult{}, err
	}

	return NewNatsConnectionResult{
		Conn: conn,
		JS:   js,
	}, nil
}

func main() {
	fx.New(
		fx.Provide(
			NewNatsConfig,
			NewNatsConnection,
		),

		fx.Invoke(func(conn *nats.Conn, js nats.JetStreamContext) {
			// second      - 1000000000
			// 30 * second - 30000000000
			// minute      - 60000000000
			// 10 * minute - 600000000000

			// Image content
			// article-main-image NewsImg
			// inside the article-main-text imgwrapper

			config := prebuiltemplate.NewsFeedConfig{
				NewsFeedURL:             "",
				NewsFeedRefreshInterval: 600000000000,
				NewsFeedArticleSelector: []string{"blog-item"},

				ArticleConfig: prebuiltemplate.ArticleConfig{
					Fields: []prebuiltemplate.Field{
						{Type: prebuiltemplate.FIELD_TYPE_TITLE, ClassSelector: "News__title"},
						{Type: prebuiltemplate.FIELD_TYPE_PREFACE, ClassSelector: "article-main-intro"},
						{Type: prebuiltemplate.FIELD_TYPE_CONTENT, ClassSelector: "article-main-text", IgnoredSentences: []string{"Отримуйте новини в Telegram", "Наші новини є у Facebook", "Дивіться нас на YouTube"}},
						{Type: prebuiltemplate.FIELD_TYPE_PUBLISHED_AT, ClassSelector: "PostInfo__item PostInfo__item_date"},
						{Type: prebuiltemplate.FIELD_TYPE_INFO, ClassSelector: "PostInfo__item PostInfo__item_service"},
					},
				},
				ArticlePrefixURL:    "https:",
				ArticlePullInterval: 30000000000,
				ArticlePageSelector: []string{"AllNewsItemInfo__name"},
			}

			newsFeed := prebuiltemplate.NewNewsFeedProcessor(config)
			go newsFeed.Start(context.Background())

			for article := range newsFeed.GetArticleChan() {
				subject := natsinfo.ArticlesStream_NewArticleSubject("test", article.Title)
				result, err := natsinfo.JsPublishJson(js, subject, article)
				log.Printf("Publish into nats %+v %+v", result, err)
			}
		}),
	)
}
