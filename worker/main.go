package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	nats "github.com/nats-io/nats.go"
	"github.com/romashorodok/news-tracker/pkg/executils"
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
	return &NatsConfig{
		Host: "nats",
		Port: "4222",
	}
}

type NewNatsConnectionResult struct {
	fx.Out

	Conn *nats.Conn
	JS   nats.JetStreamContext
}

func NewNatsConnection(config *NatsConfig) (NewNatsConnectionResult, error) {
	conn, err := nats.Connect(config.GetURL(),
		nats.Timeout(time.Second*30),
		nats.RetryOnFailedConnect(true),
	)
	if err != nil {
		return NewNatsConnectionResult{}, err
	}

	js, err := conn.JetStream()
	if err != nil {
		return NewNatsConnectionResult{}, err
	}

	wait := make(chan struct{})
	go func() {
		defer close(wait)
		ticker := time.NewTicker(time.Millisecond * 10)
		done := time.NewTimer(time.Second * 30)
		for {
			log.Printf("NATS connection state: %s", conn.Status())
			select {
			case <-done.C:
				panic("Unable establish nats connection")
			case <-ticker.C:
				if _, err = natsinfo.CreateStreamIfNotExists(js, natsinfo.ARTICLES_STREAM_CONFIG); err != nil {
					continue
				}
				return
			}
		}
	}()
	<-wait

	return NewNatsConnectionResult{
		Conn: conn,
		JS:   js,
	}, nil
}

func main() {
	<-fx.New(
		fx.Provide(
			NewNatsConfig,
			NewNatsConnection,
		),

		fx.Invoke(func(conn *nats.Conn, js nats.JetStreamContext) {
			// second      - 1000000000
			// 30 * second - 30000000000
			// minute      - 60000000000
			// 10 * minute - 600000000000
			// 30 * minute - 1800000000000

			var prebuiltemplateConfig prebuiltemplate.ConfigFlag
			flag.Var(&prebuiltemplateConfig, "template", "Enter config for parsing the source")
			flag.Parse()
			if len(prebuiltemplateConfig) == 0 {
				panic("Enter config for parsing the source by `-template` flag")
			}

			log.Printf("Running with the config: %+v", prebuiltemplateConfig)

			executils.BatchExec(prebuiltemplateConfig, len(prebuiltemplateConfig),
				func(config prebuiltemplate.NewsFeedConfig) {
					newsFeed := prebuiltemplate.NewNewsFeedProcessor(config)
					go newsFeed.Start(context.Background())

					for article := range newsFeed.GetArticleChan() {
                        origin := strings.ReplaceAll(article.Origin, ".", "_")
						subject := natsinfo.ArticlesStream_NewArticleSubject(origin, article.Title)
						result, err := natsinfo.JsPublishJson(js, subject, article)
						log.Printf("Publish into nats %+v %+v", result, err)
						log.Printf("%+v", article)
					}
				})
		}),
	).Wait()
}
