package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	chi "github.com/go-chi/chi/v5"
	_ "github.com/lib/pq"
	nats "github.com/nats-io/nats.go"
	"github.com/romashorodok/news-tracker/backend/internal/handler/v1"
	"github.com/romashorodok/news-tracker/backend/internal/service"
	"github.com/romashorodok/news-tracker/backend/internal/worker"
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
		fx.Invoke(worker.StartArticleConsumerWorker),
		fx.Invoke(StartHttpServer),
	).Run()
}
