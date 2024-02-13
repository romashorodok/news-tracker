package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	chi "github.com/go-chi/chi/v5"
	_ "github.com/lib/pq"
	nats "github.com/nats-io/nats.go"
	"github.com/romashorodok/news-tracker/backend/internal/storage"
	"github.com/romashorodok/news-tracker/pkg/dateutils"
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
	ErrArticleRequireMainImage = errors.New("article require at least the main image")
	ErrUnableCreateArticle     = errors.New("unable create the article")
	ErrUnableCreateImage       = errors.New("unable create the image")
	ErrArticleNotFound         = errors.New("article not found")
	ErrArticlesNotFound        = errors.New("articles not found")
	ErrUnableGetArticle        = errors.New("unable get article")
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

type ArticleDTO struct {
	ID            int64    `json:"id"`
	Title         string   `json:"title"`
	Preface       string   `json:"preface"`
	Content       string   `json:"content"`
	ViewersCount  int32    `json:"viewers_count"`
	PublishedAt   string   `json:"published_at"`
	MainImage     string   `json:"main_image"`
	ContentImages []string `json:"content_images,omitempty"`
}

type ArticleSorting string

var (
	ARTICLE_SORTING_NEWEST ArticleSorting = "newest"
	ARTICLE_SORTING_OLDEST ArticleSorting = "oldest"
)

type GetArticlesParams struct {
	Sorting    ArticleSorting
	StartDate  time.Time
	EndDate    time.Time
	TextLexems []string
}

var DEFAULT_START_DATE = time.Now().AddDate(-10, 0, 0)

var NULL_TIME = time.Time{}

func GetNullableSqlTime(u time.Time) sql.NullTime {
	if NULL_TIME.Equal(u) {
		return sql.NullTime{Valid: false}
	}
	return sql.NullTime{Time: u, Valid: true}
}

func (s *ArticleService) GetArticles(ctx context.Context, params GetArticlesParams) ([]ArticleDTO, error) {
	articles, err := s.queries.ArticlesWithImages(ctx, storage.ArticlesWithImagesParams{
		StartDate:        GetNullableSqlTime(params.StartDate),
		StartDateDefault: DEFAULT_START_DATE,
		EndDate:          GetNullableSqlTime(params.EndDate),
		ArticleSorting:   string(params.Sorting),
		Lexems:           params.TextLexems,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrArticlesNotFound
	}

	if len(articles) == 0 {
		return nil, ErrArticlesNotFound
	}

	var dtos []ArticleDTO

	for _, article := range articles {
		var articleImages []dbArticleImageDTO
		if err = json.Unmarshal(article.Images, &articleImages); err != nil {
			return nil, ErrUnableGetArticle
		}

		dto := ArticleDTO{
			ID:           article.ID,
			Title:        article.Title,
			Preface:      article.Preface,
			Content:      article.Content,
			ViewersCount: article.ViewersCount,
			PublishedAt:  dateutils.Pretify(article.PublishedAt),
		}

		for _, articleImage := range articleImages {
			if articleImage.Main {
				dto.MainImage = articleImage.URL
				continue
			}
			dto.ContentImages = append(dto.ContentImages, articleImage.URL)
		}

		dtos = append(dtos, dto)
	}
	return dtos, nil
}

type dbArticleImageDTO struct {
	URL  string `json:"url"`
	Main bool   `json:"main"`
}

func (s *ArticleService) GetArticleByID(ctx context.Context, id int64) (*ArticleDTO, error) {
	article, err := s.queries.GetArticleByID(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrArticleNotFound
	}
	var articleImages []dbArticleImageDTO
	if err = json.Unmarshal(article.Images, &articleImages); err != nil {
		return nil, ErrUnableGetArticle
	}

	dto := ArticleDTO{
		ID:           article.ID,
		Title:        article.Title,
		Preface:      article.Preface,
		Content:      article.Content,
		ViewersCount: article.ViewersCount,
		PublishedAt:  dateutils.Pretify(article.PublishedAt),
	}
	for _, articleImage := range articleImages {
		if articleImage.Main {
			dto.MainImage = articleImage.URL
			continue
		}
		dto.ContentImages = append(dto.ContentImages, articleImage.URL)
	}
	return &dto, err
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
	ArticleService *ArticleService
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

type ArticleHandler struct {
	articleService *ArticleService
}

var (
	SORTING_QUERY_PARAM_NAME    = "sort"
	START_DATE_QUERY_PARAM_NAME = "start_date"
	END_DATE_QUERY_PARAM_NAME   = "end_date"
	TEXT_QUERY_PARAM_NAME       = "text"
)

var ErrUnsupportedQueryParam = errors.New("")

func getDateQuery(r *http.Request, queryName string) (time.Time, error) {
	date := r.URL.Query().Get(queryName)
	if date == "" {
		return time.Time{}, nil
	}
	t, err := dateutils.ParseQueryString(date)
	if err != nil {
		return time.Time{}, errors.Join(fmt.Errorf("unsupported `%s` query value %s. Format must be like `2024-10-12T10:01`, `2024-10-12`, `YYYY-MM-DD` ", SORTING_QUERY_PARAM_NAME, r.URL.Query().Get(SORTING_QUERY_PARAM_NAME)), ErrUnsupportedQueryParam)
	}
	return t, nil
}

func getArticleSortingQuery(r *http.Request, defaultVal ArticleSorting) (ArticleSorting, error) {
	switch ArticleSorting(r.URL.Query().Get(SORTING_QUERY_PARAM_NAME)) {
	case ARTICLE_SORTING_NEWEST:
		return ARTICLE_SORTING_NEWEST, nil
	case ARTICLE_SORTING_OLDEST:
		return ARTICLE_SORTING_OLDEST, nil
	case "":
		return defaultVal, nil
	default:
		return "", errors.Join(fmt.Errorf("unsupported `%s` query value %s", SORTING_QUERY_PARAM_NAME, r.URL.Query().Get(SORTING_QUERY_PARAM_NAME)), ErrUnsupportedQueryParam)
	}
}

func getTextQuert(r *http.Request) []string {
    return strings.Split(r.URL.Query().Get(TEXT_QUERY_PARAM_NAME), " ")
}

func (hand *ArticleHandler) getArticles(w http.ResponseWriter, r *http.Request) {
	sorting, err := getArticleSortingQuery(r, ARTICLE_SORTING_NEWEST)
	if err != nil {
		articleErrHandler(w, err)
		return
	}

	startDate, err := getDateQuery(r, START_DATE_QUERY_PARAM_NAME)
	if err != nil {
		articleErrHandler(w, err)
		return
	}
	log.Println(startDate)

	endDate, err := getDateQuery(r, END_DATE_QUERY_PARAM_NAME)
	if err != nil {
		articleErrHandler(w, err)
		return
	}
	log.Println(endDate)

	textLexems := getTextQuert(r)
    log.Println(textLexems)

	// search := r.URL.Query().Get(TEXT_QUERY_PARAM_NAME)
	// log.Println(search)

	articles, err := hand.articleService.GetArticles(r.Context(), GetArticlesParams{
		Sorting:    sorting,
		StartDate:  startDate,
		EndDate:    endDate,
		TextLexems: textLexems,
	})
	if err != nil {
		articleErrHandler(w, err)
		return
	}
	json.NewEncoder(w).Encode(&articles)
}

func articleErrHandler(w http.ResponseWriter, err error) {
	switch err {
	case ErrArticlesNotFound:
		httputils.WriteErrorResponse(w, http.StatusNotFound, err.Error())
		return
	case ErrArticleNotFound:
		httputils.WriteErrorResponse(w, http.StatusNotFound, err.Error())
		return
	case ErrUnsupportedQueryParam:
		httputils.WriteErrorResponse(w, http.StatusNotAcceptable, err.Error())
	default:
		httputils.WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
	}
}

func (hand *ArticleHandler) getArticleByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		httputils.WriteErrorResponse(w, http.StatusPreconditionRequired, err.Error())
		return
	}

	article, err := hand.articleService.GetArticleByID(r.Context(), int64(id))
	if err != nil {
		articleErrHandler(w, err)
		return
	}

	json.NewEncoder(w).Encode(&article)
}

func (hand *ArticleHandler) OnRouter(router http.Handler) {
	switch r := router.(type) {
	case *chi.Mux:
		baseURL := "/api/v1"
		r.Get(baseURL+"/articles", hand.getArticles)
		r.Get(baseURL+"/articles/{id}", hand.getArticleByID)
	}
}

var _ httputils.Handler = (*ArticleHandler)(nil)

type NewArticleHandlerParams struct {
	fx.In

	ArticleService *ArticleService
}

func NewArticleHandler(params NewArticleHandlerParams) *ArticleHandler {
	return &ArticleHandler{
		articleService: params.ArticleService,
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

func main() {
	fx.New(
		fx.Provide(
			natsinfo.NewNatsConfig,
			natsinfo.NewNatsConnection,

			NewDatabaseConfig,
			NewDatabaseConnection,

			NewArticleSerivce,
			NewHttpServerConfig,

			httputils.AsHandler(groupHandler, NewArticleHandler),
		),
		// fx.Invoke(StartArticleConsumerWorker),
		fx.Invoke(StartHttpServer),
	).Run()
}
