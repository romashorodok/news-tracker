package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

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
		Host:     "localhost",
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

func NewNatsLocalhostConfig() *natsinfo.NatsConfig {
	conf := natsinfo.NewNatsConfig()
	conf.Host = "localhost"
	return conf
}

func main() {
	<-fx.New(
		fx.Provide(
			NewNatsLocalhostConfig,
			natsinfo.NewNatsConnection,

			NewDatabaseConfig,
			NewDatabaseConnection,
		),

		fx.Invoke(func(conn *sql.DB, js nats.JetStreamContext) {
			queueGroup := "backend-articles-consumer"

			if _, err := natsinfo.CreateOrUpdateStream(js, natsinfo.ARTICLES_STREAM_CONFIG); err != nil {
				log.Panicf("unable set-up nats %s stream. Err:%s", natsinfo.ARTICLES_STREAM_CONFIG.Name, err)
				os.Exit(1)
			}

			stream, subject, subOpts, config := natsinfo.ArticlesStream_NewArticleConsumerConfig(queueGroup)

			if _, err := natsinfo.CreateOrUpdateConsumer(js, stream, config); err != nil {
				log.Panicf("unable set-up nats %s consumer. Err:%s", queueGroup, err)
				os.Exit(1)
			}

			sub, err := js.QueueSubscribe(subject, queueGroup, func(msg *nats.Msg) {
				var article natsinfo.Article
				article.Unmarshal(msg.Data)
				log.Printf("%+v", article)
			}, subOpts...)

			log.Println(sub, err)

			<-context.Background().Done()

			// conn, _ := NewDatabaseConnection(NewDatabaseConnectionParams{
			// 	Config: databaseConfig,
			// })
			// defer conn.Close()

			// store := storage.New(conn)
			//
			// _, err := store.GetArticleByID(context.Background(), 0)
			// log.Println(errors.Is(err, sql.ErrNoRows))
		}),
	).Wait()
}
