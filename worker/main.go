package main

import (
	"context"
	"flag"
	"log"
	"os"
	"strings"

	nats "github.com/nats-io/nats.go"
	"github.com/romashorodok/news-tracker/pkg/executils"
	"github.com/romashorodok/news-tracker/pkg/natsinfo"
	"github.com/romashorodok/news-tracker/worker/internal/prebuiltemplate"
	"go.uber.org/fx"
)

func main() {
	<-fx.New(
		fx.Provide(
			natsinfo.NewNatsConfig,
			natsinfo.NewNatsConnection,
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

			if _, err := natsinfo.CreateOrUpdateStream(js, natsinfo.ARTICLES_STREAM_CONFIG); err != nil {
				log.Panicf("unable set-up nats %s stream. Err:%s", natsinfo.ARTICLES_STREAM_CONFIG.Name, err)
				os.Exit(1)
			}

			executils.BatchExec(prebuiltemplateConfig,
				func(config prebuiltemplate.NewsFeedConfig) {
					newsFeed := prebuiltemplate.NewNewsFeedProcessor(config)
					go newsFeed.Start(context.Background())

					for article := range newsFeed.GetArticleChan() {

						origin := strings.ReplaceAll(article.Origin, ".", "_")
						subject := natsinfo.ArticlesStream_NewArticleSubject(origin, article.Title)

						payload, err := article.Marshal()
						if err != nil {
							log.Printf("Feiled serialize article. Err: %s", err)
							log.Printf("%+v", article)
							continue
						}

						result, err := js.Publish(subject, payload)
						log.Printf("Publish into nats %+v %+v", result, err)
						log.Printf("%+v", article)
					}
				})
		}),
	).Wait()
}
